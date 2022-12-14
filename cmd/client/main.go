package main

import (
	"bufio"
	"cs.ubc.ca/cpsc416/p1/raftkv"
	"cs.ubc.ca/cpsc416/p1/util"
	"fmt"
	"github.com/DistributedClocks/tracing"
	"log"
	"os"
	"strconv"
	"strings"
)

type ClientConfig struct {
	ClientID          string
	LocalServerIPPort string
	ServerIPPortList  []string
	ChCapacity        int
	TracingServerAddr string
	Secret            []byte
	TracingIdentity   string
}

var (
	resetColour   = "\033[0m"  // default text colour
	successColour = "\033[32m" // green text
)

func main() {
	// Get server index from command line arg
	clientIdx, err := strconv.Atoi(os.Args[1])
	util.CheckErr(err, "failed to parse client index")

	filename := fmt.Sprintf("./config/client_config_%d.json", clientIdx)
	fmt.Println("Client config file:", filename)
	var config ClientConfig
	err = util.ReadJSONConfig(filename, &config)
	util.CheckErr(err, "failed to locate or parse config for client: %d\n", clientIdx)

	tracer := tracing.NewTracer(tracing.TracerConfig{
		ServerAddress:  config.TracingServerAddr,
		TracerIdentity: config.TracingIdentity,
		Secret:         config.Secret,
	})

	client := raftkv.NewKVS()
	notifCh, err := client.Start(tracer, config.ClientID, config.ServerIPPortList, config.ChCapacity)
	util.CheckErr(err, "Error reading client config: %v\n", err)

	// Setup second client if there's one
	var notifCh2 raftkv.NotifyChannel
	var client2 *raftkv.KVS
	if len(os.Args) == 4 && os.Args[3] == "-t" {
		clientIdx2, err := strconv.Atoi(os.Args[2])
		util.CheckErr(err, "failed to parse client index")

		filename := fmt.Sprintf("./config/client_config_%d.json", clientIdx2)
		fmt.Println("Client config file:", filename)
		var config ClientConfig
		err = util.ReadJSONConfig(filename, &config)
		util.CheckErr(err, "failed to locate or parse config for client: %d\n", clientIdx2)

		tracer := tracing.NewTracer(tracing.TracerConfig{
			ServerAddress:  config.TracingServerAddr,
			TracerIdentity: config.TracingIdentity,
			Secret:         config.Secret,
		})
		
		client2 = raftkv.NewKVS()
		notifCh2, err = client2.Start(tracer, config.ClientID, config.ServerIPPortList, config.ChCapacity)
		util.CheckErr(err, "Error reading client config: %v\n", err)
	}

	if len(os.Args) == 3 && os.Args[2] == "-i" {
		runInteractiveClient(client, notifCh)
	} else if len(os.Args) == 4 && os.Args[3] == "-t" {
		runTwoClientsTestScript(client, notifCh, client2, notifCh2)
	} else if len(os.Args) == 3 && os.Args[2] == "-o" {
		runTestScript(client, notifCh)
	}
}

func runTestScript(client *raftkv.KVS, notifCh raftkv.NotifyChannel) {
	// Put a key-value pair
	err := client.Put("key2", "value2")
	util.CheckErr(err, "Error putting value %v, opId: %v\b", err)

	// Get a key's value
	err = client.Get("key1")
	util.CheckErr(err, "Error getting value %v, opId: %v\b", err)

	// Sequence of interleaved gets and puts
	err = client.Put("key1", "test1")
	util.CheckErr(err, "Error putting value %v, opId: %v\b", err)
	err = client.Get("key1")
	util.CheckErr(err, "Error getting value %v, opId: %v\b", err)
	err = client.Put("key1", "test2")
	util.CheckErr(err, "Error putting value %v, opId: %v\b", err)
	err = client.Get("key1")
	util.CheckErr(err, "Error getting value %v, opId: %v\b", err)
	err = client.Get("key1")
	util.CheckErr(err, "Error getting value %v, opId: %v\b", err)
	err = client.Put("key1", "test3")
	util.CheckErr(err, "Error putting value %v, opId: %v\b", err)
	err = client.Get("key1")
	util.CheckErr(err, "Error getting value %v, opId: %v\b", err)

	for i := 0; i < 9; i++ {
		result := <-notifCh
		log.Printf("%s%v%s\n", successColour, result, resetColour)
	}
	client.Stop()
}

func runTwoClientsTestScript(client *raftkv.KVS, notifCh raftkv.NotifyChannel, client2 *raftkv.KVS, notifCh2 raftkv.NotifyChannel) {
	// Put a key-value pair
	// Expected Input: put key2 "value2"
	// Expected Output: put key2 "value2"
	err := client.Put("key2", "value2")
	util.CheckErr(err, "Error putting value %v, opId: %v\b", err)

	// Get a key's value
	// Client will get empty value for the key that has not been putted
	// Expected Input: get key1
	// Expected Output: get key1 ""
	err = client2.Get("key1")
	util.CheckErr(err, "Error getting value %v, opId: %v\b", err)

	/* Sequence of interleaved gets and puts for two clients*/

	// Client2 Put value to key1, and Client 1 can Get the value of this key1
	// Expected Input: put key1 "test1" AND get key1
	// Expected Output: put key1 "test1" AND get key1 "test1"
	err = client2.Put("key1", "test1")
	util.CheckErr(err, "Error putting value %v, opId: %v\b", err)
	err = client.Get("key1")
	util.CheckErr(err, "Error getting value %v, opId: %v\b", err)

	// Now, Client1 Put a new value to key1, and Client2 will Get the new value of key1, Client1 can Get this new value as well
	// Expected Input: put key1 "test2" AND get key1 AND get key1
	// Expected Output: put key1 "test2" AND get key1 "test2" AND get key1 "test2"
	err = client.Put("key1", "test2")
	util.CheckErr(err, "Error putting value %v, opId: %v\b", err)
	err = client2.Get("key1")
	util.CheckErr(err, "Error getting value %v, opId: %v\b", err)
	err = client.Get("key1")
	util.CheckErr(err, "Error getting value %v, opId: %v\b", err)

	// Now, Client2 Put value to a different key - key2, and Client1 can Get the value of key2 with the value just being putted
	// Expected Input: put key2 "test3" AND get key2
	// Expected Output: put key2 "test3" AND get key2 "test3"
	err = client2.Put("key2", "test3")
	util.CheckErr(err, "Error putting value %v, opId: %v\b", err)
	err = client.Get("key2")
	util.CheckErr(err, "Error getting value %v, opId: %v\b", err)

	for i := 0; i < 9; i++ {
		select {
		case result := <-notifCh:
			log.Printf("First Client:%s%v%s\n", successColour, result, resetColour)
		case result := <-notifCh2:
			log.Printf("Second Client:%s%v%s\n", successColour, result, resetColour)
		}
	}
	client.Stop()
	client2.Stop()
}

// Run client in an interactive command line
// e.g. 'put k1 v1' or 'get k1'
func runInteractiveClient(client *raftkv.KVS, notifyCh raftkv.NotifyChannel) {
	defer func() {
		client.Stop()
		log.Println("Session terminated")
	}()

	go func() {
		// Print results as they return from KVS
		for result := range notifyCh {
			log.Printf("%s%v%s\n", successColour, result, resetColour)
		}
	}()

	scanner := bufio.NewScanner(os.Stdin)
	// Parse and execute operations input by user
	for {
		scanner.Scan()
		text := scanner.Text()
		args := strings.Fields(text)
		if len(args) == 0 {
			// Terminate session on empty line
			break
		}

		op := args[0]
		if len(args) == 2 && op == "get" {
			key := args[1]
			err := client.Get(key)
			util.CheckErr(err, "Error getting value at key %s", key)
			continue
		}
		if len(args) == 3 && op == "put" {
			key := args[1]
			value := args[2]
			err := client.Put(key, value)
			util.CheckErr(err, "Error putting value %s to key %s", value, key)
			continue
		}
		log.Println("Invalid command")
	}
}
