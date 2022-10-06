/*
   Copyright (C) 2020  2ALCHEMISTS SAS.

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU Affero General Public License as
   published by the Free Software Foundation, either version 3 of the
   License, or (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU Affero General Public License for more details.

   You should have received a copy of the GNU Affero General Public License
   along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

const queryTimeLayout = "2006-01-02T15:04:05"

func createDirIfNotExists(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return os.MkdirAll(path, 0755)
	}
	return nil
}

// RoundTime rounds the given time to the provided resolution.
func RoundTime(t time.Time, resolution time.Duration) time.Time {
	return time.Unix(0, (t.UnixNano()/resolution.Nanoseconds())*resolution.Nanoseconds())
}

func getUsageHistoryPath(clusterName string) string {
	return fmt.Sprintf("%s/.usagehistory_%s", viper.GetString("krossboard_root_data_dir"), clusterName)
}

func listRegularFiles(folder string) (error, []string) {
	if _, err := os.Stat(folder); err != nil {
		return err, nil
	}
	var files []string
	err := filepath.Walk(folder, func(_ string, info os.FileInfo, err error) error {
		if !info.IsDir() && !strings.HasPrefix(info.Name(), ".") {
			files = append(files, fmt.Sprintf("%s/%s", folder, info.Name()))
		}
		return nil
	})
	return err, files
}
