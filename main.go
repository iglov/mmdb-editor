package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/maxmind/mmdbwriter"
	"github.com/maxmind/mmdbwriter/inserter"
	"github.com/maxmind/mmdbwriter/mmdbtype"
)

var (
	datasetGeo = flag.String("d", "./dataset.json", "Dataset file path.")
	inputGeo   = flag.String("i", "./GeoLite2-City.mmdb", "Input GeoLite2-City.mmdb file path.")
	outputGeo  = flag.String("o", "./GeoLite2-City-mod.mmdb", "Output modified mmdb file path.")
	version    = flag.Bool("v", false, "Print current version and exit.")
	merge      = flag.String("m", "replce", "Merge method. It may be: toplevel, recurse or replace. Default: replace")
)

var Version string

type Dataset struct {
	Dataset []Geo `json:"data"`
}

type Geo struct {
	Ips     []string `json:"ips"`
	Country Country  `json:"country"`
}

type Country struct {
	Iso_code   string `json:"iso_code"`
	Geoname_id int    `json:"geoname_id"`
	Names      Names  `json:"names"`
}

type Names struct {
	En string `json:"en"`
}

func Check(f func() error) {
	if err := f(); err != nil {
		fmt.Println("Received error:", err)
	}
}

func main() {
	// Main dataset from json
	var dataset Dataset
	// validate merge strategy.
	var mergeStrategy inserter.FuncGenerator

	flag.Parse()
	if *version {
		fmt.Println(Version)
		os.Exit(0)
	}
	if *merge == "toplevel" {
		mergeStrategy = inserter.TopLevelMergeWith
		log.Printf("Using merge method: toplevel")
	} else if *merge == "recurse" {
		mergeStrategy = inserter.DeepMergeWith
		log.Printf("Using merge method: recurse")
	} else {
		mergeStrategy = inserter.ReplaceWith
		log.Printf("Using merge method: replace")
	}

	log.Printf("Loading mmdb: %v", *inputGeo)

	// Load the database we wish to enrich.
	writer, err := mmdbwriter.Load(*inputGeo, mmdbwriter.Options{
		Inserter:                mergeStrategy,
		IncludeReservedNetworks: true,
		Description:             map[string]string{"en": fmt.Sprintf("Compiled with mmdb-editor (%v) https://github.com/iglov/mmdb-editor", Version)},
	})
	if err != nil {
		log.Fatal(err)
	}

	content, err := os.ReadFile(*datasetGeo)
	if err != nil {
		log.Fatal("Error when opening file: ", err)
	}

	err = json.Unmarshal(content, &dataset)
	if err != nil {
		log.Printf("error decoding response: %v", err)
		if e, ok := err.(*json.SyntaxError); ok {
			log.Printf("syntax error at byte offset %d", e.Offset)
		}
		log.Printf("response: %q", content)
		log.Fatal("Error during Unmarshal(): ", err)
	}

	log.Printf("Loaded data: %v", dataset.Dataset)

	for i := 0; i < len(dataset.Dataset); i++ {
		// Define and insert the new data.
		data := mmdbtype.Map{
			"country": mmdbtype.Map{
				"geoname_id": mmdbtype.Uint32(dataset.Dataset[i].Country.Geoname_id),
				"iso_code":   mmdbtype.String(dataset.Dataset[i].Country.Iso_code),
				"names": mmdbtype.Map{
					"en": mmdbtype.String(dataset.Dataset[i].Country.Names.En),
				},
			},
		}

		for _, ip := range dataset.Dataset[i].Ips {
			_, network, err := net.ParseCIDR(ip)
			if err != nil {
				log.Fatal(err)
			}
			log.Printf("Modifying net: %q", network)
			if err := writer.Insert(network, data); err != nil {
				log.Fatal(err)
			}
		}
	}

	log.Printf("Compiling and writing modified data into: %v", *outputGeo)

	// Write the newly enriched DB to the filesystem.
	fh, err := os.Create(*outputGeo)
	if err != nil {
		log.Fatal(err)
	}

	defer Check(fh.Close)

	_, err = writer.WriteTo(fh)
	if err != nil {
		log.Fatal(err)
	}
}
