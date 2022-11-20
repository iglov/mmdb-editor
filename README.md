[![Latest Release](https://img.shields.io/github/release/iglov/mmdb-editor.svg?style=flat-square)](https://github.com/iglov/mmdb-editor/releases/latest)
[![GitHub license](https://img.shields.io/github/license/iglov/mmdb-editor.svg)](https://github.com/iglov/mmdb-editor/blob/master/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/iglov/mmdb-editor)](https://goreportcard.com/report/github.com/iglov/mmdb-editor)

# mmdb-editor
Make your own GeoIP database! The simple utility for editing MMDB databases.

# How to start
1. Download lastest mmdb-editor release
2. Download [GeoLite2-City.mmdb](https://www.maxmind.com/en/accounts/current/geoip/downloads)
3. Create your dataset with networks you need to add/change (See example)

# How to use
```text
Usage of ./mmdb-editor:
  -d string
        Dataset file path. (default "./dataset.json")
  -i string
        Input GeoLite2-City.mmdb file path. (default "./GeoLite2-City.mmdb")
  -o string
        Output modified mmdb file path. (default "./GeoLite2-City-mod.mmdb")
  -v    Print current version and exit.
```

# How to develop
1. `git clone https://github.com/iglov/mmdb-editor`
2. Change something you want and commit changes
3. Build with `make all`
