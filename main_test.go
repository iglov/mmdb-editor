package main

import (
	"github.com/maxmind/mmdbinspect/pkg/mmdbinspect"
        "github.com/maxmind/mmdbwriter/mmdbtype"
        "github.com/oschwald/maxminddb-golang"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
        "bytes"
        "errors"
        "fmt"
	"os"
	"testing"
)

const (
	CityDBPathIn  = "./testdata/GeoLite2-City.testmmdb"
	CityDBPathOut = "./testdata/GeoLite2-City-mod.mmdb"
	TestDataset   = "./testdata/dataset.json"
)

func captureOutput(f func()) string {
    // Create a pipe to capture the output
    r, w, _ := os.Pipe()
    // Save the original stdout
    stdout := os.Stdout
    // Set stdout to the write end of the pipe
    os.Stdout = w

    // Run the function (which will print to stdout)
    f()

    // Close the writer end and restore stdout
    w.Close()
    os.Stdout = stdout

    // Read the captured output from the read end of the pipe
    var buf bytes.Buffer
    buf.ReadFrom(r)

    return buf.String()
}

func TestCheck(t *testing.T) {
    output := captureOutput(func() {
        Check(func() error {
            return nil
        })
    })

    // Check that there was no output (since no error should be printed)
    assert.Empty(t, output)

    output = captureOutput(func() {
        Check(func() error {
            return errors.New("test error")
        })
    })

    // Check that the error message was printed
    expectedOutput := "Received error: test error\n"
    assert.Equal(t, expectedOutput, output)
}

func TestToMMDBType(t *testing.T) {
    tests := []struct {
        key         string
        value       any
        expected    mmdbtype.DataType
        expectedErr string
    }{
        {
            key:      "bool_key",
            value:    true,
            expected: mmdbtype.Bool(true),
        },
        {
            key:      "string_key",
            value:    "test",
            expected: mmdbtype.String("test"),
        },
        {
            key: "map_key",
            value: map[string]any{
                "inner_key": "inner_value",
            },
            expected: mmdbtype.Map{
                mmdbtype.String("inner_key"): mmdbtype.String("inner_value"),
            },
        },
        {
            key: "slice_key",
            value: []any{
                "slice_value",
            },
            expected: mmdbtype.Slice{
                mmdbtype.String("slice_value"),
            },
        },
        {
            key:      "accuracy_radius",
            value:    1234.0,
            expected: mmdbtype.Uint16(1234),
        },
        {
            key:      "latitude",
            value:    52.5164,
            expected: mmdbtype.Float64(52.5164),
        },
        {
            key:         "unsupported_key",
            value:       1234.0,
            expectedErr: "unsupported numeric type for key \"unsupported_key\": float64",
        },
        {
            key:         "unsupported_type_key",
            value:       struct{}{},
            expectedErr: "unsupported type for key \"unsupported_type_key\": struct {}",
        },
    }

    for _, tt := range tests {
        t.Run(fmt.Sprintf("key=%s,value=%v", tt.key, tt.value), func(t *testing.T) {
            result, err := toMMDBType(tt.key, tt.value)

            if tt.expectedErr != "" {
                assert.Error(t, err)
                assert.Contains(t, err.Error(), tt.expectedErr)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tt.expected, result)
            }
        })
    }
}

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

	records, err := mmdbinspect.RecordsForNetwork(*reader, true, "123.125.71.29")
	a.NoError(err, "no error on lookup of 123.125.71.29")
	a.NotNil(records, "records returned")

	records, err = mmdbinspect.RecordsForNetwork(*reader, true, "127.0.0.1/32")
	a.NoError(err, "no error on lookup of 127.0.0.1/32")
	a.NotNil(records, "records returned")

	records, err = mmdbinspect.RecordsForNetwork(*reader, true, "10.200.0.33")
	a.NoError(err, "no error on lookup of 10.200.0.33")
	a.NotNil(records, "records returned")

	records, err = mmdbinspect.RecordsForNetwork(*reader, true, "192.168.33.13/30")
	a.NoError(err, "no error on lookup of 192.168.33.13/30")
	a.NotNil(records, "records returned")

	records, err = mmdbinspect.RecordsForNetwork(*reader, true, "1.1.1.1/29")
	a.NoError(err, "got no error when IP not found")
	a.Nil(records, "no records returned for 1.1.1.1/29")

	records, err = mmdbinspect.RecordsForNetwork(*reader, true, "X.X.Y.Z")
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
		records, err := mmdbinspect.RecordsForNetwork(*reader, true, ip)
		a.NoError(err, "no RecordsForNetwork error")
		prettyJSON, err := mmdbinspect.RecordToString(records)

		a.NoError(err, "no error on stringification")
		a.NotNil(prettyJSON, "records stringified")
		a.Contains(prettyJSON, "Iglov's property")
		a.Contains(prettyJSON, "6255148")
	}

	require.NoError(t, reader.Close())

}
