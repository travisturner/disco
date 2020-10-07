package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/coreos/etcd/embed"
	"github.com/coreos/etcd/etcdserver"
)

func main() {
	var (
		peer    string
		client  string
		cluster string
		name    string
		state   string
		add     bool
		seed    string
	)

	flag.StringVar(&name, "name", embed.DefaultName, "name")
	flag.StringVar(&peer, "peer", "http://localhost:10101", "peer addr")
	flag.StringVar(&client, "client", "http://localhost:20101", "client addr")
	flag.StringVar(&cluster, "cluster", "", "cluster name=addr")
	flag.StringVar(&state, "state", embed.ClusterStateFlagNew, "cluster to join")
	flag.BoolVar(&add, "add", false, "add node to cluster")
	flag.StringVar(&seed, "seed", "", "cluster seed")
	flag.Parse()

	var clusterMbrs []string

	if add {
		fmt.Println("***************** ADD NODE *********************")

		resp0, err := http.Get(seed + "/v2/members")
		if err != nil {
			log.Fatalln(err)
		}
		var result0 map[string]interface{}

		json.NewDecoder(resp0.Body).Decode(&result0)

		// map[members:[map[clientURLs:[http://localhost:20101] id:b3e8ebb5e83a520e name:default1 peerURLs:[http://localhost:10101]]]]
		mbrs, ok := result0["members"].([]interface{})
		if !ok {
			log.Fatalf("bad type: %T\n", result0["members"])
		}

		for i := range mbrs {
			mbr, ok := mbrs[i].(map[string]interface{})
			if !ok {
				log.Fatalf("bad type 2: %T\n", mbrs[i])
			}
			pname, ok := mbr["name"].(string)
			if !ok {
				log.Fatalln("bad pname")
			}
			purls, ok := mbr["peerURLs"].([]interface{})
			if !ok {
				log.Fatalln("bad purl")
			}
			purl, ok := purls[0].(string)
			if !ok {
				log.Fatalf("bad type 3: %T\n", purls[0])
			}
			clusterMbrs = append(clusterMbrs, fmt.Sprintf("%s=%s", pname, purl))
		}

		////////////////////////////
		// register member

		message := map[string]interface{}{
			"peerURLs": []string{peer},
		}

		bytesRepresentation, err := json.Marshal(message)
		if err != nil {
			log.Fatalln(err)
		}

		resp, err := http.Post(seed+"/v2/members",
			"application/json", bytes.NewBuffer(bytesRepresentation))
		if err != nil {
			log.Fatalln(err)
		}

		var result map[string]interface{}

		json.NewDecoder(resp.Body).Decode(&result)

		log.Println(result)
		fmt.Println("***************** /ADD NODE *********************")
	}

	if len(clusterMbrs) > 0 {
		cluster = strings.Join(clusterMbrs, ",")
	}

	e, err := NewEmbed(Options{
		name:       name,
		dir:        "/tmp/disco/" + name,
		clientAddr: client,
		peerAddr:   peer,
		cluster:    cluster,
		state:      state,
	})
	if err != nil {
		log.Fatal(err)
	}

	err = e.Run(func(s *etcdserver.EtcdServer) error {
		fmt.Printf("\n-----------------\nServer ID: %v, Leader: %+v, Index: %+v\n",
			s.ID(), s.Leader(), s.Index())

		c := s.Cluster()
		members := ""
		for _, member := range c.Members() {
			members += fmt.Sprintf("\n%+v", member)
		}
		fmt.Printf("Cluster ID: %v, Clients: %+v, Members: %s\n-----------------\n",
			c.ID(), c.ClientURLs(), members)
		//c.ID(), c.ClientURLs(), c.Members()[0])

		time.Sleep(2 * time.Second)
		return nil
	})

	log.Fatal(err)
}
