package guru

import (
	"context"
	"fmt"

	// "log"
	b64 "encoding/base64"
	"regexp"
	"strings"

	"net/http"

	"github.com/trufflesecurity/trufflehog/pkg/common"
	"github.com/trufflesecurity/trufflehog/pkg/detectors"
	"github.com/trufflesecurity/trufflehog/pkg/pb/detectorspb"
)

type Scanner struct{}

// Ensure the Scanner satisfies the interface at compile time
var _ detectors.Detector = (*Scanner)(nil)

var (
	client = common.SaneHttpClient()

	//Make sure that your group is surrounded in boundry characters such as below to reduce false positives
	unamePat = regexp.MustCompile(detectors.PrefixRegex([]string{"guru"}) + `\b([a-zA-Z0-9]{3,20}@[a-zA-Z0-9]{2,12}.[a-zA-Z0-9]{2,5})\b`)
	keyPat   = regexp.MustCompile(detectors.PrefixRegex([]string{"guru"}) + `\b([a-z0-9]{8}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{12})\b`)
)

// Keywords are used for efficiently pre-filtering chunks.
// Use identifiers in the secret preferably, or the provider name.
func (s Scanner) Keywords() []string {
	return []string{"guru"}
}

// FromData will find and optionally verify Guru secrets in a given set of bytes.
func (s Scanner) FromData(ctx context.Context, verify bool, data []byte) (results []detectors.Result, err error) {
	dataStr := string(data)

	unameMatches := unamePat.FindAllStringSubmatch(dataStr, -1)
	keyMatches := keyPat.FindAllStringSubmatch(dataStr, -1)

	for _, match := range unameMatches {
		if len(match) != 2 {
			continue
		}

		unameMatch := strings.TrimSpace(match[1])

		for _, secret := range keyMatches {
			if len(secret) != 2 {
				continue
			}

			keyMatch := strings.TrimSpace(secret[1])

			s1 := detectors.Result{
				DetectorType: detectorspb.DetectorType_Guru,
				Raw:          []byte(unameMatch),
			}

			if verify {
				data := fmt.Sprintf("%s:%s", unameMatch, keyMatch)
				encoded := b64.StdEncoding.EncodeToString([]byte(data))

				req, _ := http.NewRequest("GET", "https://api.getguru.com/api/v1/teams/teamId/stats", nil)
				req.Header.Add("Authorization", fmt.Sprintf("Basic %s", encoded))
				res, err := client.Do(req)
				if err == nil {
					defer res.Body.Close()
					if res.StatusCode >= 200 && res.StatusCode < 300 {
						s1.Verified = true
					} else {
						//This function will check false positives for common test words, but also it will make sure the key appears 'random' enough to be a real key
						if detectors.IsKnownFalsePositive(keyMatch, detectors.DefaultFalsePositives, true) {
							continue
						}
					}
				}
			}

			results = append(results, s1)
		}
	}

	return detectors.CleanResults(results), nil
}
