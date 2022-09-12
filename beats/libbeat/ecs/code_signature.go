// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package ecs

import (
	"time"
)

// These fields contain information about binary code signatures.
type CodeSignature struct {
	// Boolean to capture if a signature is present.
	Exists bool `ecs:"exists"`

	// Subject name of the code signer
	SubjectName string `ecs:"subject_name"`

	// Boolean to capture if the digital signature is verified against the
	// binary content.
	// Leave unpopulated if a certificate was unchecked.
	Valid bool `ecs:"valid"`

	// Stores the trust status of the certificate chain.
	// Validating the trust of the certificate chain may be complicated, and
	// this field should only be populated by tools that actively check the
	// status.
	Trusted bool `ecs:"trusted"`

	// Additional information about the certificate status.
	// This is useful for logging cryptographic errors with the certificate
	// validity or trust status. Leave unpopulated if the validity or trust of
	// the certificate was unchecked.
	Status string `ecs:"status"`

	// The team identifier used to sign the process.
	// This is used to identify the team or vendor of a software product. The
	// field is relevant to Apple *OS only.
	TeamID string `ecs:"team_id"`

	// The identifier used to sign the process.
	// This is used to identify the application manufactured by a software
	// vendor. The field is relevant to Apple *OS only.
	SigningID string `ecs:"signing_id"`

	// The hashing algorithm used to sign the process.
	// This value can distinguish signatures when a file is signed multiple
	// times by the same signer but with a different digest algorithm.
	DigestAlgorithm string `ecs:"digest_algorithm"`

	// Date and time when the code signature was generated and signed.
	Timestamp time.Time `ecs:"timestamp"`
}
