//go:build reference && !testhooks

package main

import "github.com/JiangZhan-s/ArenaLab/internal/reference/server"

func installHooks(*server.Service) (func(), error) { return func() {}, nil }
