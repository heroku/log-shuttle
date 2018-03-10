/**
 * Copyright (c) 2018 Salesforce
 * All rights reserved.
 * Licensed under the BSD 3-Clause license.
 * For full license text, see LICENSE.txt file in the repo root
 *   or https://opensource.org/licenses/BSD-3-Clause
 */

package shuttle

import "time"

// LogLine holds the new line terminated log messages and when shuttle received them.
type LogLine struct {
	line []byte
	when time.Time
}

// Length returns the length of the raw byte of the LogLine
func (ll LogLine) Length() int {
	return len(ll.line)
}
