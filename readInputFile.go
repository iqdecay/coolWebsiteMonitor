package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	url2 "net/url"
	"os"
	"strconv"
	"strings"
)

// Represents the parameters for one of the monitored website
type WebsiteParameter struct {
	domain   string // Website to check, has to be a valid url
	interval int    // Time between checks, in milliseconds
}

// Get the parameters for website monitoring, put them into an array of structs
// and return it
// The file format should be as follows :
// url1 interval1
// url2 interval2
// where interval{1,2} are integers in ms, and url{1,2} are valid urls
func getParameters() []WebsiteParameter {
	// Implementation could be optimized by first reading the size of the input file
	var parameters []WebsiteParameter
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
	nLine := 1
	log.Printf("Reading %s ...", *filename)
	for s.Scan() {
		line := strings.Trim(s.Text(), " ")
		splitLine := strings.Split(line, " ")
		if len(splitLine) != 2 {
			log.Fatalf("Reading %s line %d: expected 2 words found %d",
				*filename, nLine, len(splitLine))
		}
		// Check url validity
		webParam := WebsiteParameter{}
		_, err := url2.ParseRequestURI(splitLine[0])
		if err != nil {
			log.Fatalf("Converting from %s line %d : invalid url in first argument '%s'",
				*filename, nLine, splitLine[0])
		}
		webParam.domain = splitLine[0]
		interval, err := strconv.Atoi(splitLine[1])
		if err != nil {
			log.Fatalf("Converting from %s line %d : found non-int type for second argument '%s'",
				*filename, nLine, splitLine[1])
		}
		webParam.interval = interval
		parameters = append(parameters, webParam)
		nLine++
	}
	err = s.Err()
	if err := s.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading standard input:", err)
	}
	log.Printf("Done")
	return parameters
}
