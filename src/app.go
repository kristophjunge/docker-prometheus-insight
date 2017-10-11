package main

import (
    "io"
    "net/http"
    "log"
    "os"
    "strconv"
    "io/ioutil"
    "encoding/json"
    "errors"
)

const LISTEN_ADDRESS = ":9208"

var config Config

type Config []struct {
    ApiUrl string `json:"apiUrl"`
    AccountId string `json:"accountId"`
}

type InsightData struct {
    Balance float64 `json:"balance"`
}

func integerToString(value int64) string {
    return strconv.FormatInt(value, 10)
}

func floatToString(value float64, precision int) string {
    return strconv.FormatFloat(value, 'f', precision, 64)
}

func formatValue(key string, meta string, value string) string {
    result := key;
    if (meta != "") {
        result += "{" + meta + "}";
    }
    result += " "
    result += value
    result += "\n"
    return result
}


func getConfig() (Config, error) {
    dir, err := os.Getwd()
    if err != nil {
        log.Fatal(err)
    }
    body, err := ioutil.ReadFile(dir + "/config.json")
    if err != nil {
        log.Fatal(err)
    }

    bodyStr := string(body)

    jsonData := Config{}
    json.Unmarshal([]byte(bodyStr), &jsonData)

    return jsonData, nil
}

func queryData(apiUrl string, accountId string) (string, error) {
    // Build URL
    url := apiUrl + "/" + accountId

    // Perform HTTP request
    resp, err := http.Get(url);
    if err != nil {
        return "", err;
    }

    // Parse response
    defer resp.Body.Close()
    if resp.StatusCode != 200 {
        return "", errors.New("HTTP returned code " + integerToString(int64(resp.StatusCode)))
    }
    bodyBytes, err := ioutil.ReadAll(resp.Body)
    bodyString := string(bodyBytes)
    if err != nil {
        return "", err;
    }

    return bodyString, nil;
}

func metrics(w http.ResponseWriter, r *http.Request) {
    log.Print("Serving /metrics")

    var up int64
    var jsonString string
    var err error

    for _, miner := range config {
        up = 1

        // Query miner statistics
        jsonString, err = queryData(miner.ApiUrl, miner.AccountId)
        if err != nil {
            log.Print(err)
            up = 0
        }

        // Parse JSON
        jsonData := InsightData{}
        json.Unmarshal([]byte(jsonString), &jsonData)

        /*
        // Check response status
        if (jsonData.Status != "OK") {
            log.Print("Received negative status in JSON response '" + jsonData.Status + "'")
            log.Print(jsonString)
            up = 0
        }
        */

        // Output
        io.WriteString(w, formatValue("insight_up", "apiUrl=\"" + miner.ApiUrl + "\",account=\"" + miner.AccountId + "\"", integerToString(up)))
        io.WriteString(w, formatValue("insight_balance", "apiUrl=\"" + miner.ApiUrl + "\",account=\"" + miner.AccountId + "\"", floatToString(jsonData.Balance, 19)))
    }
}

func index(w http.ResponseWriter, r *http.Request) {
    log.Print("Serving /index")
    html := `<!doctype html>
<html>
    <head>
        <meta charset="utf-8">
        <title>Insight Exporter</title>
    </head>
    <body>
        <h1>Insight Exporter</h1>
        <p><a href="/metrics">Metrics</a></p>
    </body>
</html>`
    io.WriteString(w, html)
}

func main() {
    var err error
    config, err = getConfig()
    if err != nil {
        log.Fatal(err)
    }

    log.Print("Insight exporter listening on " + LISTEN_ADDRESS)
    http.HandleFunc("/", index)
    http.HandleFunc("/metrics", metrics)
    http.ListenAndServe(LISTEN_ADDRESS, nil)
}
