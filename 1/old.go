package main

import (
  "fmt"
  "time"
  "net"
  "bufio"
  "os"
  "io"
  "strconv"
  "strings"
  "sync"
  "encoding/gob"
)

type NNode struct {
  id string
  ip string
  port string
  conn net.Conn
}

type NMap struct {
  sync.Mutex
  m map[string]*NNode
}

func newNMap() NMap {
  var nm NMap
  nm.m = make(map[string]*NNode)
  return nm
}

func (nm NMap) add(n *NNode) bool {
  if _, ok := nm.m[n.id]; !ok {
    nm.m[n.id] = n
    return true
  }
  return false
}

func (nm NMap) sendToAll() {
  for _, n := range nm.m {
    if haveBothConns(n.id) {
      fmt.Println("Sending to: " + n.id)
      enc := gob.NewEncoder(n.conn)
      MY_DV.printDV()
      err := enc.Encode(MY_DV)
      if err != nil {
        fmt.Println("FROM ENCODING:")
        fmt.Println(err)
      }
    } else {
      fmt.Println("Cannot send to: " + n.id)
    }
  }
}

type DV struct {
  Owner string
  M map[string]*NodeInfo
}

func newDV() DV {
  var dv DV
  dv.Owner = ID
  dv.M = make(map[string]*NodeInfo)
  return dv
}

func (dv DV) add(s string) {
  if ni, ok := dv.M[s]; !ok {
    dv.M[s] = &NodeInfo{s, 1}
  } else {
    ni.Next_hop = s
    ni.Hops_to = 1
  }
}

func (dv DV) update(ndv *DV) bool {
  changed := false
  for id, ni := range ndv.M {
    if id != ID {
      my_entry, ok := dv.M[id]
      if !ok {
        dv.M[id] = &NodeInfo{ndv.Owner, ni.Hops_to + 1}
        changed = true
      } else {
        if my_entry.Hops_to > (ni.Hops_to + 1) {
          my_entry.Next_hop = ndv.Owner
          my_entry.Hops_to = ni.Hops_to + 1
          changed = true
        }
      }
    }
  }
  return changed
}

func (dv DV) printDV() {
  //fmt.Println("UPDATED NODE " + dv.Owner + " ROUTING TABLE")
  //fmt.Println("TO NODE  |  NEXT HOP |  HOPS TO")
  for id, ni := range dv.M {
    fmt.Println("    " + id + "\t        " + ni.Next_hop + "\t    " + strconv.Itoa(ni.Hops_to))
  }
}

type NodeInfo struct {
  Next_hop string
  Hops_to int
}

var ID string
var IP string
var PORT string

var TO_NEIGH NMap
var FROM_NEIGH NMap
var MY_DV DV

func main() {
  nl := initialize()

  TO_NEIGH = newNMap() //only send DV through this map
  FROM_NEIGH = newNMap()
  MY_DV = newDV()

  centralStation(nl)
}

func centralStation(nl *[]string) {
  dialed := make(chan *NNode)
  recieved := make(chan *NNode)
  addToDV := make(chan string)
  neighbourDV := make(chan *DV)

  for _, s := range(*nl) {
    go dial(s, dialed)
  }
  go listen(recieved)

  for {
    select {
    case n := <-dialed:
      if TO_NEIGH.add(n) {
        if haveBothConns(n.id) {
          go func() {addToDV <- n.id}()
        }
      } else {
        fmt.Println("WHOA SOMETHING WENT WRONG. I TRIED TO DIAL SOMETHING I ALREADY HAVE")
      }
    case n := <-recieved:
      if FROM_NEIGH.add(n) {
        go getNeighData(n, neighbourDV)
        if haveBothConns(n.id) {
          go func() {addToDV <- n.id}()
        } else {
          s := colonize(n)
          if needToDial(s, nl) {
            go dial(s, dialed)
          }
        }
      } else {
        fmt.Println("WHOA SOMETHING WENT WRONG. I LISTENED TO SOMETHING I ALREADY HAVE")
      }
    case s := <-addToDV:
      MY_DV.add(s)
      TO_NEIGH.sendToAll()
      //MY_DV.printDV()
    case v := <-neighbourDV:
      if MY_DV.update(v) {
        TO_NEIGH.sendToAll()
        //MY_DV.printDV()
      }
    }
  }
}

func listen(recieved chan<- *NNode) {
  ln, err := net.Listen("tcp", ":" + PORT)
  if err != nil {
    fmt.Println(err)
    return
  }
  for {
    c, err := ln.Accept()
    if err != nil {
      fmt.Println(err)
      continue
    }
    scanner := bufio.NewScanner(c)
    scanner.Scan()
    info := scanner.Text()
    ip, port, id := uncolonize(info)
    nnode := &NNode{id, ip, port, c}
    recieved <- nnode
  }
}

func getNeighData(n *NNode, neighbourDV chan<- *DV) {
  for {
    var ndv DV
    dec := gob.NewDecoder(n.conn)
    err := dec.Decode(&ndv)
    if err != nil {
      if err == io.EOF {
        fmt.Println("Closed connection from: \t" + n.id)
        n.conn.Close()
        break
      } else {
        fmt.Println("Recieved a bad DV from: " + n.id)
        fmt.Println(err)
      }
    } else {
      fmt.Println("Recieved a good DV from: " + n.id)
      neighbourDV <- &ndv

    }
  }
}


func dial(s string, dialed chan<- *NNode) {
  test := true
  ip, port, id := uncolonize(s)
  for test {
    dial, err := net.Dial("tcp", ip + ":" + port)
    if err != nil {
      time.Sleep(3 * time.Second)
      continue
    }
    fmt.Fprint(dial, colonizeMe() + "\n")
    nnode :=  &NNode{id, ip, port, dial}
    dialed <- nnode
    test = false
  }
}

//Helper Functions
func haveBothConns(id string) bool {
  _, okL := FROM_NEIGH.m[id]
  _, okD := TO_NEIGH.m[id]

  return (okL && okD)
}

func needToDial(s string, nl *[]string) bool {
  for i:= 0; i < len(*nl); i++ {
    if s == (*nl)[i] {
      return false
    }
  }
  fmt.Println("I NEED TO DIAL " + s)
  return true
}

//String Processing
func uncolonize(s string) (string, string, string) {
  sArr := strings.Split(s, ":")
  return sArr[0], sArr[1], sArr[2]
}

func colonizeMe() (string) {
  return IP + ":" + PORT + ":" + ID
}

func colonize(n *NNode) (string) {
  return n.ip+":"+n.port+":" + n.id
}

//Data Intialization Functions
func readNeighbours(scanner *bufio.Scanner) (*[]string) {
  var lines []string
  for scanner.Scan() {
    lines = append(lines, scanner.Text())
  }
  return &lines
}

func initialize() (*[]string) {
  file, _ := os.Open("neighbours.add")
  defer file.Close()

  scanner := bufio.NewScanner(file)
  scanner.Scan()
  IP, PORT, ID = uncolonize(scanner.Text())

  nl := readNeighbours(scanner)

  return nl
}
