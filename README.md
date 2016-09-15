# antifrost
[![License](http://img.shields.io/:license-gpl3-blue.svg?style=flat-square)](http://www.gnu.org/licenses/gpl-3.0.html)
[![GitHub release](https://img.shields.io/github/release/tulir/antifrost.svg?style=flat-square)](https://github.com/tulir/antifrost/releases)

A program wrapper that will kill and restart programs if they freeze.

## Compiling
Install the [Go toolkit](https://golang.org/doc/install) and use `go get -u maunium.net/go/antifrost`

## Basic usage
It's recommended to put antifrost flags first, then two dashes to instruct [mauflag](https://github.com/tulir/mauflag) to stop looking for flags and finally the command without extra quoting.

Use `antifrost -h` to see antifrost flag usage.
