package handlers

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"github.com/igorsilva-dev/crashdummy/app/models"
	"net/http"
	"net/url"
	"os"
	"time"
)

func getaMapping(name string) models.Mapping {

	jsonFile, err := os.Open("mappings/" + name)

	if err != nil {
		fmt.Println(err)
	}

	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	var mapping models.Mapping

	json.Unmarshal(byteValue, &mapping)

	stubFile, err := os.Open("stubs/" + mapping.Response.BodyFileName)

	if err != nil {
		fmt.Println(err)
	}

	defer stubFile.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(stubFile)
	content := buf.String()

	mapping.MappedResponse = content

	return mapping
}

func getProxy(name string) models.Proxy {

	jsonFile, err := os.Open("proxies/" + name)

	if err != nil {
		fmt.Println(err)
	}

	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	var proxy models.Proxy

	json.Unmarshal(byteValue, &proxy)

	return proxy
}

func read_mappings() []models.Mapping {

	var mappings []models.Mapping

	files, err := ioutil.ReadDir("./mappings")

	if err != nil {
		log.Fatal(err)
	}

	for _, f := range files {
		mapping := getaMapping(f.Name())
		mappings = append(mappings, mapping)
	}

	return mappings
}

func read_proxies() []models.Proxy {

	var proxies []models.Proxy

	files, err := ioutil.ReadDir("./proxies")

	if err != nil {
		log.Fatal(err)
	}

	for _, f := range files {
		proxy := getProxy(f.Name())
		proxies = append(proxies, proxy)
	}

	return proxies
}

func mappRequest(mapping models.Mapping) {

	http.HandleFunc(mapping.Request.Url, func(w http.ResponseWriter, r *http.Request) {

		c := make(chan map[string]interface{})

		go func(chan map[string]interface{}) {
			var response map[string]interface{}
			json.Unmarshal([]byte(mapping.MappedResponse), &response)
			c <- response
		}(c)

		x := <-c
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Chaos-Type", "mock")
		json.NewEncoder(w).Encode(x)

	})
}

func createHttpClient(proxyEnabled bool) *http.Client {

	tlsConfig := &tls.Config{}
	tlsConfig.InsecureSkipVerify = true

	if proxyEnabled {

		proxyUrl, err := url.Parse("http://proxy01:8080")

		if err != nil {
			fmt.Println("Proxy Error!", err)
		}

		return &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyUrl), TLSClientConfig: tlsConfig, MaxIdleConns: 30, MaxIdleConnsPerHost: 30}}
	}

	return &http.Client{Transport: &http.Transport{TLSClientConfig: tlsConfig, MaxIdleConns: 30, MaxIdleConnsPerHost: 30}}

}

func mappProxy(proxy models.Proxy) {

	http.HandleFunc(proxy.Path, func(w http.ResponseWriter, r *http.Request) {

		c := make(chan interface{})

		go func(chan interface{}) {

			jitter := rand.Int63n(int64(proxy.JitterInMillieconds*2)) - int64(proxy.JitterInMillieconds)

			time.Sleep(time.Duration(proxy.LatencyInMillieconds+int(jitter)) * time.Millisecond)

			httpClient := createHttpClient(false)

			req, err := http.NewRequest(proxy.Method, proxy.Upstream, nil)
			req.Close = true

			if err != nil {
				log.Println(err)
			}

			response, err := httpClient.Do(req)

			if err != nil {
				log.Println(err)
			}

			responseData, err := ioutil.ReadAll(response.Body)
			response.Body.Close()

			var res map[string]interface{}
			json.Unmarshal(responseData, &res)

			if err != nil {
				log.Println(err)
			}

			c <- res

		}(c)

		x := <-c
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Chaos-Type", "proxy")
		json.NewEncoder(w).Encode(x)

	})
}

func Initiate() {

	mappings := read_mappings()

	for _, m := range mappings {
		mappRequest(m)
	}

	proxies := read_proxies()

	for _, m := range proxies {
		mappProxy(m)
	}

}
