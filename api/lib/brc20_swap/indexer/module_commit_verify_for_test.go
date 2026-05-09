package indexer

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

func InitResultDataFromFile(fname string) (err error) {
	// Open our jsonFile
	jsonFile, err := os.Open(fname)
	// if we os.Open returns an error then handle it
	if err != nil {
		fmt.Println(err)
		return err
	}
	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()

	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return err
	}
	err = json.Unmarshal([]byte(byteValue), &GResultsExternal)
	if err != nil {
		return err
	}
	return nil
}
