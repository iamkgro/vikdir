package main

import (
	"gopkg.in/pin/tftp.v1"

	"bufio"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
)

// Can be netascii, octet, mail (octet is relevant here)
var mode = "octet"

// DirURL struct to hold the initial parsed config URL
type DirURL struct {
	DirectoryURL string `xml:"directoryURL"`
}

// URLQuery struct to hold the list of input URL values
type URLQuery struct {
	MenuItems []MenuItem `xml:"MenuItem"`
}

// MenuItem struct of individual input URL values
type MenuItem struct {
	Name string
	URL  string
}

// CorpListQuery struct to hold directory listing URL value
type CorpListQuery struct {
	URL string
}

// PaginateQuery struct to grab Next pagination URL
type PaginateQuery struct {
	SoftKeyItems []SoftKeyItem `xml:"SoftKeyItem"`
}

// SoftKeyItem struct to hold the pagination values
type SoftKeyItem struct {
	Name string
	URL  string
}

// CorpEntryQuery struct to hold Name/Phone values
type CorpEntryQuery struct {
	CorpEntries []CorpEntry `xml:"DirectoryEntry"`
}

// CorpEntry struct holds each Name/Phone value
type CorpEntry struct {
	Name      string
	Telephone string
}

func main() {
	hostname := flag.String("hostname", "", "the hostname of a phone (e.g., SEP12345678)")
	server := flag.String("server", "", "the ip-address or hostname of the tftp server")
	port := flag.String("port", "", "the tftp port (default is set to 69)")
	flag.Parse()

	if *hostname == "" {
		log.Fatal("a phone handset hostname is required")
	}
	if *server == "" {
		log.Fatal("an IP or hostname of the tftp server is required")
	}
	if *port == "" {
		*port = "69"
	}

	var filename = *hostname + ".cnf.xml"

	tftpget(*hostname, *server, *port, filename)
	localeURL := getlocaledir(filename)
	inputURL := getinputdir(localeURL)
	listURL := getlistdir(inputURL)
	getcorplist(listURL)

}

func tftpget(hostname, server, port, filename string) {
	addr, err := net.ResolveUDPAddr("udp", server+":"+port)
	if err != nil {
		fmt.Println(err)
	}
	file, err := os.Create(filename)
	if err != nil {
		fmt.Println(err)
	}
	w := bufio.NewWriter(file)
	log := log.New(os.Stderr, "", log.Ldate|log.Ltime)
	c := tftp.Client{addr, log}
	c.Get(filename, mode, func(reader *io.PipeReader) {
		n, readError := w.ReadFrom(reader)
		if readError != nil {
			fmt.Fprintf(os.Stderr, "Can't get %s: %v\n", filename, readError)
			log.Fatal(readError)
		} else {
			fmt.Fprintf(os.Stderr, "Got %s (%d bytes)\n", filename, n)
		}
		w.Flush()
		file.Close()
	})
}

func getlocaledir(filename string) string {
	xmlData, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't get %s: %v\n", filename, xmlData)
		log.Fatal(err)
	}
	var d DirURL
	xml.Unmarshal(xmlData, &d)
	return d.DirectoryURL
}

func getinputdir(localeURL string) string {
	var inputURL string

	res, err := http.Get(localeURL)
	if err != nil {
		fmt.Printf("Request timeout to %s", localeURL)
		log.Fatal(err)
	} else {
		defer res.Body.Close()
		contents, err := ioutil.ReadAll(res.Body)
		if err != nil {
			fmt.Printf("%s", err)
			log.Fatal(err)
		}
		var u URLQuery
		xml.Unmarshal(contents, &u)
		for _, v := range u.MenuItems {
			if v.Name == "Corporate Directory" {
				inputURL = v.URL
			}
		}
	}
	return inputURL
}

func getlistdir(inputURL string) string {
	var listURL string

	res, err := http.Get(inputURL)
	if err != nil {
		fmt.Printf("Request timeout to %s", inputURL)
		log.Fatal(err)
	} else {
		defer res.Body.Close()
		contents, err := ioutil.ReadAll(res.Body)
		if err != nil {
			fmt.Printf("%s", err)
			log.Fatal(err)
		}
		var c CorpListQuery
		xml.Unmarshal(contents, &c)
		listURL = c.URL
	}
	return listURL
}

func getcorplist(listURL string) {
	var pageURL string

	res, err := http.Get(listURL)
	if err != nil {
		fmt.Printf("Request timeout to %s", listURL)
		log.Fatal(err)
	} else {
		defer res.Body.Close()
		contents, err := ioutil.ReadAll(res.Body)
		if err != nil {
			fmt.Printf("%s", err)
			log.Fatal(err)
		}

		var e CorpEntryQuery
		xml.Unmarshal(contents, &e)
		for _, v := range e.CorpEntries {
			fmt.Println("****Account****")
			fmt.Printf("Name: %s\n", v.Name)
			fmt.Printf("Telephone: %s\n", v.Telephone)
		}

		var p PaginateQuery
		xml.Unmarshal(contents, &p)
		for _, v := range p.SoftKeyItems {
			if v.Name == "Next" {
				pageURL = v.URL
				getcorplist(pageURL)
			}
		}
	}
}
