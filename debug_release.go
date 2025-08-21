//go:build !debug
// +build !debug

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

// debugLog is a no-op when not compiled with -tags debug
func debugLog(format string, v ...interface{}) {}

// debugLogln is a no-op when not compiled with -tags debug
func debugLogln(v ...interface{}) {}

// debugPrint is a no-op when not compiled with -tags debug
func debugPrint(format string, v ...interface{}) {}