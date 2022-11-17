package main

import (
	"log"
	"net"
	"os"
	"encoding/json"
	"io/ioutil"

	"github.com/maxmind/mmdbwriter"
	"github.com/maxmind/mmdbwriter/inserter"
	"github.com/maxmind/mmdbwriter/mmdbtype"
)

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

func main() {
        var dataset Dataset
        const mmdb = "GeoLite2-City.mmdb"

        log.Printf("Loading mmdb: %v", mmdb)

	// Load the database we wish to modify.
	writer, err := mmdbwriter.Load(mmdb, mmdbwriter.Options{
                IncludeReservedNetworks: true,
                Description: map[string]string {"en": "Compiled with love by iglov (c) https://github.com/iglov/mmdb-editor"},
        })
	if err != nil { log.Fatal(err) }

        content, err := ioutil.ReadFile("dataset.json")
        if err != nil {
                log.Fatal("Error when opening file: ", err)
        }

        err = json.Unmarshal(content, &dataset)
        if err != nil {
                log.Printf("error decoding sakura response: %v", err)
                if e, ok := err.(*json.SyntaxError); ok {
                        log.Printf("syntax error at byte offset %d", e.Offset)
                }
                log.Printf("sakura response: %q", content)
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
                        log.Printf("Compiling net: %q", network)
                	if err != nil { log.Fatal(err) }
                        // We can use here inserter.TopLevelMergeWith but we need to replace data with new one
	                if err := writer.InsertFunc(network, inserter.ReplaceWith(data)); err != nil {
		                log.Fatal(err)
        	        }

                }
        }

	// Write the newly enriched DB to the filesystem.
	fh, err := os.Create("GeoLite2-City-mod.mmdb")
	if err != nil { log.Fatal(err) }

        defer fh.Close()

	_, err = writer.WriteTo(fh)
	if err != nil { log.Fatal(err) }
}
