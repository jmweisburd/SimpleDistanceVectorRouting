Julian Weisburd
Advanced Distributed Systems
Lab 3 
22/02/2017

This is the README for my LAB 3 of Advanced Disbtributed Systems.
This code simulates a simple Distance-vector routing algorithm

HOW TO RUN:

In the directory named lab3_U141863 there are 9 subdirectories:

1 2 3 4 5 6 7 8 9

Each directory corresponds to a node in the example network on the Lab Session 3 project specification sheet.
Directory 1 is the node with ID 1, directory 2 is the node with ID 2 etc.

To start a node type:

go run node.go

into a terminal. 
In order to simulate the routing table algorithm on the whole network, open up 9 terminals, change into a nodes subdirectory and start the node.

Whenever a node becomes aware of a change in its routing table it will print out the updated routing table in its terminal.

HOW TO ADD A NODE:

To add a node to the network, a new directory with should be made in the same directory as the rest of the node directories. Then, you must copy of all the files in an older directory into the new node directory. The new node needs a unique ID and needs to be listening on a unique port as well as a known IP. This can be achieved by editing the first line of the "neighbours.add" file in the form of: 

IP:PORT:ID

where IP is the IP of the new node, PORT is the listening port of the new node, and ID is the unique ID. 

The subsequent lines in the "neighbours.add" file should be edited to the IP, port, and IDs of the neighbours that the new node wishes to connect to.
The neighbour nodes that should be connected to have the same syntax:

IP:PORT:ID
