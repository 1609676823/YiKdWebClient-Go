package main

import (
	"flag"
	"fmt"
	"log"

	yikdwebclient "gitee.com/lnsyzjw/yi-kd-web-client-go"
)

func main() {
	configPath := flag.String("config", "YiKdWebCfg/appsettings.xml", "appsettings.xml path")
	flag.Parse()

	client, err := yikdwebclient.NewClientFromConfig(*configPath)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	payload := `{
  "FormId": "BD_Customer",
  "FieldKeys": "FCUSTID,FNumber,FName",
  "FilterString": [],
  "OrderString": "",
  "TopRowCount": 10,
  "StartRow": 0,
  "Limit": 10
}`
	response, err := client.ExecuteBillQuery(payload)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(response)
}
