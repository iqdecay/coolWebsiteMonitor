package main

import (
	"bufio"
	"flag"
	"log"
	urlpkg "net/url"
	"os"
	"strings"
	"time"
)

// UrlWatchParameters holds parameters for a url watched by the monitor
type UrlWatchParameters struct {
	url      string        // Website to check, has to be a valid url
	interval time.Duration // Time between checks
}

// Get the parameters for website monitoring, put them into an array of structs
// and return it
// The file format should be as follows :
// url1 interval1
// url2 interval2
// where interval{1,2} are duration (see https://golang.org/pkg/time/#ParseDuration)
// and url{1,2} are valid urls
func parseParameterFile() []UrlWatchParameters {
	// Implementation could be optimized by first reading the size of the input file
	var parameters []UrlWatchParameters
	filename := flag.String("f", "websites.txt", "file path to read from")
	flag.Parse()

	f, err := os.Open(*filename)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err = f.Close(); err != nil {
			log.Fatal(err)
		}
	}()
	s := bufio.NewScanner(f)
	log.Printf("Reading %s ...", *filename)
	nLine := 1
	for s.Scan() {
		urlWatchParams := extractUrlWatchParameterFromLine(s.Text(), nLine, *filename)
		parameters = append(parameters, urlWatchParams)
		nLine++
	}
	if err := s.Err(); err != nil {
		log.Fatalf("Reading standard input: %v", err)
	}
	log.Printf("Done")
	return parameters
}

func extractUrlWatchParameterFromLine(s string, nLine int, filename string) UrlWatchParameters {
	line := strings.Trim(s, " ")
	splitLine := strings.Split(line, " ")
	if len(splitLine) != 2 {
		log.Fatalf("Reading %s line %d: expected 2 words found %d",
			filename, nLine, len(splitLine))
	}
	urlString, intervalString := splitLine[0], splitLine[1]
	_, err := urlpkg.ParseRequestURI(urlString)
	if err != nil {
		log.Fatalf("Converting from %s line %d : invalid url in first argument '%s'",
			filename, nLine, urlString)
	}
	interval, err := time.ParseDuration(intervalString)
	if err != nil {
		log.Fatalf("Converting from %s line %d: %v",
			filename, nLine, err)
	}
	return UrlWatchParameters{urlString, interval}
}
