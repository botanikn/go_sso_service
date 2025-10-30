package main

import (
	"fmt"

	"github.com/botanikn/go_sso_service/internal/config"
)

func main() {

	cfg := config.MustLoad()

	fmt.Println(cfg)

	// Logger

	// SSO Service

}