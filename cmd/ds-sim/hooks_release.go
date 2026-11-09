//go:build reference && !testhooks

package main

import (
	"github.com/JiangZhan-s/ArenaLab/internal/reference/dedicated"
	"net/http"
)

func installTestRoutes(*http.ServeMux, *dedicated.Runtime) {}
