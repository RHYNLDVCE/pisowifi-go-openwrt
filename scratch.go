package main

import (
    "fmt"
    paho "github.com/eclipse/paho.mqtt.golang"
)

func main() {
    opts := paho.NewClientOptions()
    fmt.Printf("Default order: %v\n", opts.Order)
}
