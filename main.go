package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"reflect"

	"github.com/maxmind/mmdbwriter"
	"github.com/maxmind/mmdbwriter/inserter"
	"github.com/maxmind/mmdbwriter/mmdbtype"
)

var (
	datasetGeo = flag.String("d", "./dataset.json", "Dataset file path.")
	inputGeo   = flag.String("i", "./GeoLite2-City.mmdb", "Input GeoLite2-City.mmdb file path.")
	outputGeo  = flag.String("o", "./GeoLite2-City-mod.mmdb", "Output modified mmdb file path.")
	version    = flag.Bool("v", false, "Print current version and exit.")
	merge      = flag.Bool("m", false, "Merge Mode")
)

var Version string = "1.0.3"

type Dataset struct {
	Dataset []Geo `json:"data"`
}

type Geo struct {
	Ips     []string `json:"ips"`
	Country Country  `json:"country"`
	City    City     `json:"city"`
	Org     string   `json:"org"`
}

type Country struct {
	Iso_code   string `json:"iso_code"`
	Geoname_id int    `json:"geoname_id"`
	Names      Names  `json:"names"`
}
type City struct {
	Geoname_id int   `json:"geoname_id"`
	Names      Names `json:"names"`
}

type Names struct {
	En string `json:"en"`
	Ru string `json:"ru"`
}

func Empty(val interface{}) bool {
	v := reflect.ValueOf(val)
	switch v.Kind() {
	case reflect.String, reflect.Array:
		return v.Len() == 0
	case reflect.Map, reflect.Slice:
		return v.Len() == 0 || v.IsNil()
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	}

	return reflect.DeepEqual(val, reflect.Zero(v.Type()).Interface())
}

func Check(f func() error) {
	if err := f(); err != nil {
		fmt.Println("Received error:", err)
	}
}

func main() {
	var dataset Dataset

	flag.Parse()
	if *version {
		fmt.Println(Version)
		os.Exit(0)
	}

	log.Printf("Loading mmdb: %v", *inputGeo)

	// Load the database we wish to enrich.
	writer, err := mmdbwriter.Load(*inputGeo, mmdbwriter.Options{
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
					"ru": mmdbtype.String(dataset.Dataset[i].Country.Names.Ru),
				},
			},
			"city": mmdbtype.Map{
				"geoname_id": mmdbtype.Uint32(dataset.Dataset[i].City.Geoname_id),
				"names": mmdbtype.Map{
					"en": mmdbtype.String(dataset.Dataset[i].City.Names.En),
					"ru": mmdbtype.String(dataset.Dataset[i].City.Names.Ru),
				},
			},
		}
		if Empty(dataset.Dataset[i].Org) == false {
			data["org"] = mmdbtype.String(dataset.Dataset[i].Org)
		}

		for _, ip := range dataset.Dataset[i].Ips {
			_, network, err := net.ParseCIDR(ip)
			log.Printf("Modifying net: %q", network)
			if err != nil {
				log.Fatal(err)
			}
			// We can use here inserter.TopLevelMergeWith but we need to replace data with new one
			if *merge {
				if err := writer.InsertFunc(network, inserter.TopLevelMergeWith(data)); err != nil {
					log.Fatal(err)
				}
			} else {
				if err := writer.InsertFunc(network, inserter.ReplaceWith(data)); err != nil {
					log.Fatal(err)
				}
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
