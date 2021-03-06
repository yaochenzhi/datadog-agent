// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2018 Datadog, Inc.

// +build !windows

package flare

import (
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

func zipCounterStrings(tempDir, hostname string) error {
	return nil
}

func zipTypeperfData(tempDir, hostname string) error {
	return nil
}

// Add puts the given filepath in the map
// of files to process later during the commit phase.
func (p permissionsInfos) add(filePath string) {
	p[filePath] = filePermsInfo{}
}

// Commit resolves the infos of every stacked files in the map
// and then writes the permissions.log file on the filesystem.
func (p permissionsInfos) commit(tempDir, hostname string, mode os.FileMode) error {
	if err := p.statFiles(); err != nil {
		return err
	}
	if err := p.write(tempDir, hostname, mode); err != nil {
		return err
	}
	return nil
}

func (p permissionsInfos) statFiles() error {
	for filePath := range p {
		fi, err := os.Stat(filePath)
		if err != nil {
			log.Println(err)
			return fmt.Errorf("while getting info of %s: %s", filePath, err)
		}

		sys, ok := fi.Sys().(*syscall.Stat_t)
		if !ok {
			// not enough information to append for this file
			// might rarely happen on system not supporting this feature, but as
			// we're building with !windows tag, shouldn't happen except for plan9
			return fmt.Errorf("can't retrieve file uid/gid infos")
		}

		u, err := user.LookupId(strconv.Itoa(int(sys.Uid)))
		if err != nil {
			return fmt.Errorf("can't lookup for uid info: %v", err)
		}
		g, err := user.LookupGroupId(strconv.Itoa(int(sys.Gid)))
		if err != nil {
			return fmt.Errorf("can't lookup for gid info: %v", err)
		}

		p[filePath] = filePermsInfo{
			mode:  fi.Mode(),
			owner: u.Name,
			group: g.Name,
		}
	}
	return nil
}

func (p permissionsInfos) write(tempDir, hostname string, mode os.FileMode) error {
	// init the file
	t := filepath.Join(tempDir, hostname, "permissions.log")

	if err := ensureParentDirsExist(t); err != nil {
		return err
	}

	f, err := os.OpenFile(t, os.O_RDWR|os.O_CREATE|os.O_APPEND, mode)
	if err != nil {
		return fmt.Errorf("while opening: %s", err)
	}

	defer f.Close()

	// write headers
	s := fmt.Sprintf("%-50s | %-5s | %-10s | %-10s\n", "File path", "mode", "owner", "group")
	if _, err = f.Write([]byte(s)); err != nil {
		return err
	}
	if _, err = f.Write([]byte(strings.Repeat("-", len(s)) + "\n")); err != nil {
		return err
	}

	// write each file permissions infos
	for filePath, perms := range p {
		_, err = f.WriteString(fmt.Sprintf("%-50s | %-5s | %-10s | %-10s\n", filePath, perms.mode.String(), perms.owner, perms.group))
		if err != nil {
			return err
		}
	}

	return nil
}
