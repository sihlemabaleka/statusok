package database

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/influxdb/influxdb/client"
	"log"
	"net/url"
	"os"
	"time"
)

const (
	MyHost        = "localhost"
	MyPort        = 8086
	DatabaseName  = "statusok"
	MyMeasurement = "shapes"
)

var (
	influxDBcon *client.Client
)

func DatabaseInit() {
	u, err := url.Parse(fmt.Sprintf("http://%s:%d", MyHost, MyPort))
	if err != nil {
		log.Fatal(err)
	}

	conf := client.Config{
		URL:      *u,
		Username: os.Getenv("INFLUX_USER"),
		Password: os.Getenv("INFLUX_PWD"),
	}

	influxDBcon, err = client.NewClient(conf)
	if err != nil {
		log.Fatal(err)
	}

	dur, ver, err := influxDBcon.Ping()
	if err != nil {
		log.Fatal(err)
	}
	//createDbErr := createDatabase(DatabaseName)
	//if createDbErr != nil {
	//return error
	//}
	log.Printf("Happy as a Hippo! %v, %s", dur, ver)
}

func createDatabase(databaseName string) error {

	_, err := queryDB(fmt.Sprintf("create database %s", databaseName))

	return err
}

func WritePoints(url string, responseTime int64, requestType string) {

	var pts = make([]client.Point, 0)
	point := client.Point{
		Measurement: url,
		Tags: map[string]string{
			"requestType":  requestType,
			"responseTime": "responseTime",
		},
		Fields: map[string]interface{}{
			"responseTime": responseTime,
			"errorReason":  "Not Found ",
			"errorMessage": "this the response we got",
		},
		Time:      time.Now(),
		Precision: "ms",
	}

	pts = append(pts, point)

	bps := client.BatchPoints{
		Points:          pts,
		Database:        DatabaseName,
		RetentionPolicy: "default",
	}

	_, err := influxDBcon.Write(bps)

	if err != nil {
		log.Fatal(err)
	}
}
func queryDB(cmd string) (res []client.Result, err error) {
	q := client.Query{
		Command:  cmd,
		Database: DatabaseName,
	}
	if response, err := influxDBcon.Query(q); err == nil {
		if response.Error() != nil {
			return res, response.Error()
		}
		res = response.Results
	}
	return
}

func GetMeanResponseTime(Url string, span int) (float64, error) {
	q := fmt.Sprintf(`select mean(responseTime) from "%s" WHERE time > now() - %dm GROUP BY time(%dm)`, Url, span, span)
	res, err := queryDB(q)
	if err != nil {
		fmt.Println(err)
		return 0, err
	}

	//Retrive the last record
	noOfRows := len(res[0].Series[0].Values)
	fmt.Println(q)
	if noOfRows != 0 {
		row := res[0].Series[0].Values[noOfRows-1]
		t, err := time.Parse(time.RFC3339, row[0].(string))
		if err != nil || row[1] == nil {

			fmt.Println("error ", err, " ", row[1])
			return 0, err
		}
		val, err2 := row[1].(json.Number).Float64()
		if err2 != nil {

			fmt.Println(err)
			return 0, err2
		}

		fmt.Println("[%2d] %s: %03d\n", 1, t.Format(time.Stamp), val, err2)
		return val, nil
	}
	return 0, errors.New("error")
}

func GetErrorsCount(Url string, span int) (int64, error) {
	//TODO:fix the value insde count for errors
	q := fmt.Sprintf(`select count(responseTime) from "%s" WHERE time > now() - %dm GROUP BY time(%dm)`, Url, span, span)

	res, err := queryDB(q)

	if err != nil {
		log.Fatal(err)
		return 0, err
	}

	count := res[0].Series[0].Values[len(res[0].Series[0].Values)-1][1]
	fmt.Println("Found a total of records", count)
	if count == nil {

		return 0, err
	}
	value, convErr := count.(json.Number).Int64()

	if convErr != nil {
		return 0, convErr
	}

	return value, nil
}