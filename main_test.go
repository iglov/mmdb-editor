package main

import (
	"github.com/maxmind/mmdbinspect/pkg/mmdbinspect"
	"github.com/oschwald/maxminddb-golang"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

const (
	CityDBPathIn  = "./testdata/GeoLite2-City.testmmdb"
	CityDBPathOut = "./testdata/GeoLite2-City-mod.mmdb"
	TestDataset   = "./testdata/dataset.json"
)

func TestMain(t *testing.T) {
	os.Args = []string{"mmdb-editor",
		"-i", CityDBPathIn,
		"-o", CityDBPathOut,
		"-d", TestDataset}
	main()
}

func TestOpenDB(t *testing.T) {
	a := assert.New(t)

	a.FileExists(CityDBPathIn, "database exists")

	reader, err := mmdbinspect.OpenDB(CityDBPathIn)
	a.NoError(err, "no open error")
	a.IsType(maxminddb.Reader{}, *reader)

	reader, err = mmdbinspect.OpenDB("foo/bar/baz")
	a.Error(err, "open error when file does not exist")
	a.Nil(reader)
	a.Equal(
		"foo/bar/baz does not exist",
		err.Error(),
	)

	reader, err = mmdbinspect.OpenDB(TestDataset)
	a.Error(err)
	a.Contains(err.Error(), "could not be opened: error opening database: invalid MaxMind DB file")
	a.Nil(reader)

	if reader != nil {
		require.NoError(t, reader.Close())
	}
}

func TestRecordsForNetwork(t *testing.T) {
	a := assert.New(t)
	reader, err := mmdbinspect.OpenDB(CityDBPathOut) // ipv6 database
	a.NoError(err, "no open error")

	records, err := mmdbinspect.RecordsForNetwork(*reader, "123.125.71.29")
	a.NoError(err, "no error on lookup of 123.125.71.29")
	a.NotNil(records, "records returned")

	records, err = mmdbinspect.RecordsForNetwork(*reader, "127.0.0.1/32")
	a.NoError(err, "no error on lookup of 127.0.0.1/32")
	a.NotNil(records, "records returned")

	records, err = mmdbinspect.RecordsForNetwork(*reader, "10.200.0.33")
	a.NoError(err, "no error on lookup of 10.200.0.33")
	a.NotNil(records, "records returned")

	records, err = mmdbinspect.RecordsForNetwork(*reader, "192.168.33.13/30")
	a.NoError(err, "no error on lookup of 192.168.33.13/30")
	a.NotNil(records, "records returned")

	records, err = mmdbinspect.RecordsForNetwork(*reader, "1.1.1.1/29")
	a.NoError(err, "got no error when IP not found")
	a.Nil(records, "no records returned for 1.1.1.1/29")

	records, err = mmdbinspect.RecordsForNetwork(*reader, "X.X.Y.Z")
	a.Error(err, "got an error")
	a.Nil(records, "no records returned for X.X.Y.Z")
	a.Equal("X.X.Y.Z is not a valid IP address", err.Error())

	require.NoError(t, reader.Close())
}

func TestRecordToString(t *testing.T) {
	a := assert.New(t)
	ips := []string{"10.200.0.0/24", "10.200.0.211/24", "10.200.0.89", "192.168.33.13/30", "192.168.33.14", "127.0.0.1/32"}

	reader, err := mmdbinspect.OpenDB(CityDBPathOut)
	a.NoError(err, "no open error")

	for _, ip := range ips {
		records, err := mmdbinspect.RecordsForNetwork(*reader, ip)
		a.NoError(err, "no RecordsForNetwork error")
		prettyJSON, err := mmdbinspect.RecordToString(records)

		a.NoError(err, "no error on stringification")
		a.NotNil(prettyJSON, "records stringified")
		a.Contains(prettyJSON, "Iglov's property")
		a.Contains(prettyJSON, "6255148")
	}

	require.NoError(t, reader.Close())

}
