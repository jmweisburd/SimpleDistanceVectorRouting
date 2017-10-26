package main

import (
  "fmt"
  "time"
  "net"
  "bufio"
  "os"
  //"strconv"
  "strings"
  "sync"
  //"encoding/gob"
)

var ID string
var IP string
var PORT string

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

func newNMap() *NMap {
  var nm NMap
  nm.m = make(map[string]*NNode)
  return &nm
}

func main() {
  //nm := newNMap()
  nl := initialize()
  connectionSetup(nl)
}

func connectionSetup(nl *[]string) {
  fromDial := make(chan *NNode)
  fromListen := make(chan *NNode)

  for _, n := range(*nl) {
    go dial(n, fromDial)
  }

  go listen(fromListen)

  for {
    select {
    case n := <-fromDial:
      fmt.Println(n.ip, n.port, n.id)
    case n := <-fromListen:
      fmt.Println(n.ip, n.port, n.id)
    }
  }
}

func listen(fromListen chan<- *NNode) {
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
    fromListen <- nnode
  }
}


func dial(s string, fromDial chan<- *NNode) {
  test := true
  ip, port, id := uncolonize(s)
  for test {
    dial, err := net.Dial("tcp", ip + ":" + port)
    if err != nil {
      fmt.Println(err)
      time.Sleep(1000 * time.Millisecond)
      continue
    }
    fmt.Fprint(dial, colonizeMe() +"\n")
    nnode := &NNode{id, ip, port, dial}
    test = false
    fromDial <- nnode
  }
  fmt.Println("CONNECTION DIALED")
}

//String Processing
func uncolonize(s string) (string, string, string) {
  sArr := strings.Split(s, ":")
  return sArr[0], sArr[1], sArr[2]
}

func colonizeMe() (string) {
  return IP + ":" + PORT + ":" + ID
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
