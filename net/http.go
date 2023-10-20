package http

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net/http"
	"time"
)

func Get(url string) ([]byte, error) {
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	rep, err := client.Get(url)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(rep.Body)
	rep.Body.Close()
	if err != nil {
		return nil, err
	}
	switch {
	case rep.StatusCode >= 400:
		return nil, errors.New(http.StatusText(rep.StatusCode))
	}
	return body, nil

}

func Post(url string, requestBody []byte) ([]byte, error) {
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	rep, err := client.Post(url, "application/json", bytes.NewReader(requestBody))
	if err != nil {
		return nil, err
	}
	defer rep.Body.Close()
	body, err := ioutil.ReadAll(rep.Body)
	if err != nil {
		return nil, err
	}

	switch {
	case rep.StatusCode >= 400:
		return nil, errors.New(http.StatusText(rep.StatusCode))
	}
	return body, nil
}

func PostForm(url string, keyName string, requestBody []byte) ([]byte, error) {
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	data := []byte(keyName + "=")
	data = append(data, requestBody...)
	rep, err := client.Post(url, "application/x-www-form-urlencoded", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer rep.Body.Close()
	body, err := ioutil.ReadAll(rep.Body)

	if err != nil {
		return nil, err
	}

	switch {
	case rep.StatusCode >= 400:
		return nil, errors.New(http.StatusText(rep.StatusCode))
	}
	return body, nil
}
