/*******************************************************************************
*   (c) Zondax AG
*
*  Licensed under the Apache License, Version 2.0 (the "License");
*  you may not use this file except in compliance with the License.
*  You may obtain a copy of the License at
*
*      http://www.apache.org/licenses/LICENSE-2.0
*
*  Unless required by applicable law or agreed to in writing, software
*  distributed under the License is distributed on an "AS IS" BASIS,
*  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
*  See the License for the specific language governing permissions and
*  limitations under the License.
********************************************************************************/

package ledger_go

import (
	"os"
	"strings"

	"github.com/zondax/golem/pkg/logger"
)

var log *logger.Logger

func init() {
	initLogger()
}

func initLogger() {
	level := getLogLevel()
	config := logger.Config{
		Level:    level,
		Encoding: "console",
	}

	log = logger.NewLogger(config)
}

func getLogLevel() string {
	level := os.Getenv("LEDGER_LOG_LEVEL")
	if level == "" {
		level = "info" //default to info
	}
	return strings.ToLower(level)
}
