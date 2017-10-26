package main

import (
  "fmt"
  "time"
  "net"
  "bufio"
  "os"
  "strconv"
  "strings"
)


//Structure specifing Node data
type NNode struct {
  id string //unique node identifer
  ip string //ip of node
  port string //port node listens and recieves on
  conn net.Conn
}

//Structure containing map of node ids to node data
type NMap struct {
  m map[string]*NNode
}

func newNMap() NMap {
  var nm NMap
  nm.m = make(map[string]*NNode)
  return nm
}

/*
Add a new node to a node map
Input: NNode
Output: true if input NNode added, false if not
*/
func (nm NMap) add(n *NNode) bool {
  if _, ok := nm.m[n.id]; !ok {
    nm.m[n.id] = n
    return true
  }
  return false
}

/*
Send this node's distance vector to all neighboring nodes in a node map
*/
func (nm NMap) sendToAll() {
  s := stringifyDV(MY_DV) //make this node's distance vector into a string
  for _, n := range nm.m {
    fmt.Fprint(n.conn, s)
  }
}


//Structure describing a distance vector
type DV struct {
  Owner string //ID of node who owns the distance vector
  M map[string]*NodeInfo //Map of nodes to information about the nodes
}

func newDV() DV {
  var dv DV
  dv.Owner = ID
  dv.M = make(map[string]*NodeInfo)
  return dv
}

/*
Add a new entry to the distance vector map
Only used when a neighbor directly connects to this node
Input: Node ID (string)
Output:
*/
func (dv DV) add(s string) {
  if ni, ok := dv.M[s]; !ok {
    dv.M[s] = &NodeInfo{s, 1}
  } else {
    ni.Next_hop = s
    ni.Hops_to = 1
  }
}

/*
Update a local distance vector (this node's) after recieving a distance vector from a neighboring node
Input: Stringified distance vector
Output: True if this node's distance vector changed, false if otherwise
*/
func (dv DV) update(s string) bool {
  changed := false
  prep := strings.Split(s, ",,")
  dv_owner := prep[0] //Who owns the recieved distance vector
  entries := prep[1:] //All entries in the recieved distance vector
  for _, v := range entries { //loop through all entries in the recieved distance vector
    pre := strings.Split(v, ",")
    n_id := pre[0]
    n_ht, _ := strconv.Atoi(pre[2])
    if n_id != ID {
      my_entry, ok := dv.M[n_id]
      if !ok { //If this node doesn't know how to get to entry node
        dv.M[n_id] = &NodeInfo{dv_owner, n_ht+1} //Add new node to this node's distance vector
        changed = true
      } else {
        //This node know's how to get to entry node but the recieved distance vector's way is shorter
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

//Print out a distance vector to the terminal
func (dv DV) printDV() {
  fmt.Println("UPDATED NODE " + dv.Owner + " ROUTING TABLE")
  fmt.Println("TO NODE  |  NEXT HOP |  HOPS TO")
  for id, ni := range dv.M {
    fmt.Println("    " + id + "\t        " + ni.Next_hop + "\t    " + strconv.Itoa(ni.Hops_to))
  }
  fmt.Println("")
}

//Structure describing how to get to a node and how many hops it takes to get to node
type NodeInfo struct {
  Next_hop string
  Hops_to int
}

//This node's ID,IP,and PORT
var ID string
var IP string
var PORT string

var TO_NEIGH NMap //Connections to send to neighbor nodes
var FROM_NEIGH NMap //Connections to recieve from neighbor nodes
var MY_DV DV //This node's distance vector

func main() {
  nl := initialize()

  TO_NEIGH = newNMap() //only send DV through this map
  FROM_NEIGH = newNMap()
  MY_DV = newDV()

  centralStation(nl)
}

//Function that handles 
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
    case n := <-dialed: //A dialed node has accepted the connection
      if TO_NEIGH.add(n) { //Add to the dialed node map
        if haveBothConns(n.id) {
          go func() {addToDV <- n.id}()
        }
      } else {
        fmt.Println("WHOA SOMETHING WENT WRONG. I TRIED TO DIAL SOMETHING I ALREADY HAVE")
      }
    case n := <-recieved: //A neighbor node has dialed us and we accepted the connection
      if FROM_NEIGH.add(n) {
        go getNeighData(n, neighbourDV)
        if haveBothConns(n.id) {
          go func() {addToDV <- n.id}()
        } else { //If we accepted a connection but don't have a corresponding dialed connection
          s := colonize(n)
          if needToDial(s, nl) {
            go dial(s, dialed)
          }
        }
      } else {
        fmt.Println("WHOA SOMETHING WENT WRONG. I LISTENED TO SOMETHING I ALREADY HAVE")
      }
    case s := <-addToDV: //When we have both dialed/recieved conns to a node, add that node to our DV
      MY_DV.add(s)
      TO_NEIGH.sendToAll()
      MY_DV.printDV()
    case v := <-neighbourDV: //We received a DV from a neighboring node
      if MY_DV.update(v) {
        TO_NEIGH.sendToAll()
        MY_DV.printDV()
      }
    }
  }
}

/*
Go routine used to listen for new connections
Input: channel to send new Node structures through
Output:
*/
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
    ip, port, id := uncolonize(info) //parse information about neighbor node
    nnode := &NNode{id, ip, port, c} //make new node and pass through channel
    recieved <- nnode
  }
}


/*
Go routine used to listen for DV sent through accepted connections
Input: Node structure to listen to, channel to pass DV through
Output: 
*/
func getNeighData(n *NNode, neighbourDV chan<- string) {
  scanner := bufio.NewScanner(n.conn)
  for scanner.Scan() {
    dv := scanner.Text()
    neighbourDV <- dv
  }
  fmt.Println("Connection from " + n.id + " closed.")
  n.conn.Close()
}


/*
Go routine used to dial neighbor nodes 
Input: String describing node to dial, channel to pass dialed node structure through
Output:
*/
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
