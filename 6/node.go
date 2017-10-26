package main

import (
  "fmt"
  "time"
  "net"
  "bufio"
  "os"
  "strconv"
  "strings"
  "sync"
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
  s := stringifyDV(MY_DV)
  for _, n := range nm.m {
    fmt.Fprint(n.conn, s)
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

func (dv DV) update(s string) bool {
  changed := false
  prep := strings.Split(s, ",,")
  dv_owner := prep[0]
  entries := prep[1:]
  for _, v := range entries {
    pre := strings.Split(v, ",")
    n_id := pre[0]
    n_ht, _ := strconv.Atoi(pre[2])
    if n_id != ID {
      my_entry, ok := dv.M[n_id]
      if !ok {
        dv.M[n_id] = &NodeInfo{dv_owner, n_ht+1}
        changed = true
      } else {
        if my_entry.Hops_to > n_ht + 1 {
          my_entry.Next_hop = dv_owner
          my_entry.Hops_to = n_ht + 1
          changed = true
        }
      }
    }
  }
  return changed
}

func (dv DV) printDV() {
  fmt.Println("UPDATED NODE " + dv.Owner + " ROUTING TABLE")
  fmt.Println("TO NODE  |  NEXT HOP |  HOPS TO")
  for id, ni := range dv.M {
    fmt.Println("    " + id + "\t        " + ni.Next_hop + "\t    " + strconv.Itoa(ni.Hops_to))
  }
  fmt.Println("")
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
  neighbourDV := make(chan string)

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
      MY_DV.printDV()
    case v := <-neighbourDV:
      if MY_DV.update(v) {
        TO_NEIGH.sendToAll()
        MY_DV.printDV()
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


func getNeighData(n *NNode, neighbourDV chan<- string) {
  scanner := bufio.NewScanner(n.conn)
  for scanner.Scan() {
    dv := scanner.Text()
    neighbourDV <- dv
  }
  fmt.Println("Connection from " + n.id + " closed.")
  n.conn.Close()
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

func stringifyDV(dv DV) string {
  s :=  dv.Owner
  for id, ni := range dv.M {
    s = s + ",," + id + "," + ni.Next_hop + "," + strconv.Itoa(ni.Hops_to)
  }
  s = s + "\n"
  return s
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
