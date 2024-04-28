package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go4.org/netipx"
	"log"
	"net/netip"
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
	merge      = flag.String("m", "replace", "Merge strategy. It may be: toplevel, recurse or replace.")
)

// Version contains main version of build. Get from compiler variables
var Version string

// Dataset is for JSON dataset mapping
type Dataset struct {
	Networks []string       `json:"networks"`
	Data     map[string]any `json:"data"`
}

// Check func for defer functions
func Check(f func() error) {
	if err := f(); err != nil {
		fmt.Println("Received error:", err)
	}
}

// toMMDBType key converts field values read from json into their corresponding mmdbtype.DataType.
// It makes some assumptions for numeric types based on previous knowledge about field types.
func toMMDBType(key string, value any) (mmdbtype.DataType, error) {
	switch v := value.(type) {
	case bool:
		return mmdbtype.Bool(v), nil
	case string:
		return mmdbtype.String(v), nil
	case map[string]any:
		m := mmdbtype.Map{}
		for innerKey, val := range v {
			innerVal, err := toMMDBType(innerKey, val)
			if err != nil {
				return nil, fmt.Errorf("parsing mmdbtype.Map for key %q: %w", key, err)
			}
			m[mmdbtype.String(innerKey)] = innerVal
		}
		return m, nil
	case []any:
		s := mmdbtype.Slice{}
		for _, val := range v {
			innerVal, err := toMMDBType(key, val)
			if err != nil {
				return nil, fmt.Errorf("parsing mmdbtype.Slice for key %q: %w", key, err)
			}
			s = append(s, innerVal)
		}
		return s, nil
	case float64:
		switch key {
		case "accuracy_radius", "confidence", "metro_code":
			return mmdbtype.Uint16(v), nil
		case "autonomous_system_number", "average_income",
			"geoname_id", "ipv4_24", "ipv4_32", "ipv6_32",
			"ipv6_48", "ipv6_64", "population_density":
			return mmdbtype.Uint32(v), nil
		case "ip_risk", "latitude", "longitude", "score",
			"static_ip_score":
			return mmdbtype.Float64(v), nil
		default:
			return nil, fmt.Errorf("unsupported numeric type for key %q: %T", key, value)
		}
	default:
		return nil, fmt.Errorf("unsupported type for key %q: %T", key, value)
	}
}

func main() {
	// main data map for json dataset
	var dataset []Dataset
	// validate merge strategy.
	var mergeStrategy inserter.FuncGenerator

	flag.Parse()
	if *version {
		fmt.Println(Version)
		os.Exit(0)
	}
	if *merge == "toplevel" {
		mergeStrategy = inserter.TopLevelMergeWith
		log.Printf("Using merge strategy: toplevel")
	} else if *merge == "recurse" {
		mergeStrategy = inserter.DeepMergeWith
		log.Printf("Using merge strategy: recurse")
	} else {
		mergeStrategy = inserter.ReplaceWith
		log.Printf("Using merge strategy: replace")
	}

	log.Printf("Loading mmdb: %v", *inputGeo)

	// Load the database we wish to enrich.
	dbWriter, err := mmdbwriter.Load(*inputGeo, mmdbwriter.Options{
		Inserter:                mergeStrategy,
		IncludeReservedNetworks: true,
		Description:             map[string]string{"en": fmt.Sprintf("Compiled with mmdb-editor (%v) https://github.com/iglov/mmdb-editor", Version)},
	})
	if err != nil {
		log.Fatal(err)
	}

	file, err := os.Open(*datasetGeo)
	if err != nil {
		log.Fatal("Opening json file error:", err)
	}
	defer Check(file.Close)

	if err := json.NewDecoder(file).Decode(&dataset); err != nil {
		log.Printf("error decoding response: %v", err)
		if e, ok := err.(*json.SyntaxError); ok {
			log.Printf("syntax error at byte offset %d", e.Offset)
		}
		log.Printf("response: %v", file)
		log.Fatal("Error during Unmarshal(): ", err)
	}

	for _, record := range dataset {
		for _, network := range record.Networks {
			prefix, err := netip.ParsePrefix(network)
			if err != nil {
				log.Fatal("Parsing networks error:", err)
			}

			mmdbValue, err := toMMDBType(prefix.String(), record.Data)
			if err != nil {
				log.Fatal("Converting value to mmdbtype error:", err)
			}

			log.Printf("Modifying net: %s", prefix.String())
			if err := dbWriter.Insert(netipx.PrefixIPNet(prefix), mmdbValue); err != nil {
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

	_, err = dbWriter.WriteTo(fh)
	if err != nil {
		log.Fatal(err)
	}
}
