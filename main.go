package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type Geometry struct {
	Coordinates []float32 `json:"coordinates"`
}

type Properties struct {
	Image    string  `json:"image"`
	Dop      string  `json:"dop"`
	TotalSat string  `json:"total_sat"`
	Datetime string  `json:"datetime"`
	Long     float32 `json:"long"`
	Lat      float32 `json:"lat"`
	Alt      float32 `json:"alt"`
}

type Properties1 struct {
	Name string `json:"name"`
}

type Feature struct {
	Id           string     `json:"id"`
	Geometry     Geometry   `json:"geometry"`
	GeometryName string     `json:"geometry_name"`
	Properties   Properties `json:"properties"`
}

type Base struct {
	TotalFeatures uint        `json:"totalFeatures"`
	Features      []Feature   `json:"features"`
	Crs           Properties1 `json:"crs"`
}

func main() {
	var totalFotos uint
	var fotosBaixadas uint

	jsonData := new(Base)
	err := getJsonOfImagesFromIbama(jsonData)
	if err != nil {
		log.Fatal(err)
	}
	for _, v := range jsonData.Features {
		dir := dirPathFromUrl(v.Properties.Image)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			err := os.Mkdir(dir, os.ModePerm)
			if err != nil {
				log.Fatal(err)
			}
		}
		totalFotos++
	}

	var wg sync.WaitGroup
	errs := make(chan error, len(jsonData.Features))
	sem := make(chan struct{}, 80)

	for _, v := range jsonData.Features {
		wg.Add(1)
		go func(feature Feature) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			err := downloadImagefromUrl(feature.Properties.Image)
			if err != nil {
				errs <- err
			} else {
				fotosBaixadas++
			}
			fmt.Println(fotosBaixadas, "/", totalFotos)
		}(v)
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			log.Println("Download error:", err)
		}
	}

	fmt.Println("Imagens Baixadas")
}

func getJsonOfImagesFromIbama(target interface{}) error {
	resp, err := http.Get("http://siscom.ibama.gov.br/geoserver/publica/ows?service=WFS&version=1.0.0&request=GetFeature&typeName=publica:img_foto_rio_doce_p&outputFormat=application%2Fjson")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(target)
	if err != nil {
		return err
	}
	return nil
}

func dirPathFromUrl(url string) string {
	nameSuffixed := strings.TrimPrefix(url, "http://siscom.ibama.gov.br/imgmariana/")
	namePlusDir := strings.TrimSuffix(nameSuffixed, filepath.Ext(nameSuffixed))
	dirPath, _ := filepath.Split(namePlusDir)
	return dirPath
}

func downloadImagefromUrl(url string) error {
	nameSuffixed := strings.TrimPrefix(url, "http://siscom.ibama.gov.br/imgmariana/")
	namePlusDir := strings.TrimSuffix(nameSuffixed, filepath.Ext(nameSuffixed))
	file, err := os.Create(namePlusDir + ".jpg")
	if err != nil {
		return err
	}
	defer file.Close()
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return err
	}
	return nil
}
