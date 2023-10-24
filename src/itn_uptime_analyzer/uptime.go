package itn_uptime_analyzer

import (
    dg "block_producers_uptime/delegation_backend"
    "encoding/json"
    "io"
    "regexp"
    "strconv"
    "strings"
    "time"

    "github.com/aws/aws-sdk-go-v2/service/s3"
    logging "github.com/ipfs/go-log/v2"
)

// This function calculates the difference between the time elapsed today and the execution interval, decides if it need to check multiple buckets or not and calculates the uptime
func (identity Identity) GetUptime(config AppConfig, ctx dg.AwsContext, log *logging.ZapEventLogger, syncPeriod int) {

    day := config.Period.Start.Format("2006-01-02")
    numberOfSubmissionsNeeded := (60 / syncPeriod) * int(config.Period.Interval.Hours())

    prefixToday := strings.Join([]string{ctx.Prefix, "submissions", day}, "/")

    inputToday := &s3.ListObjectsV2Input{
        Bucket: ctx.BucketName,
        Prefix: &prefixToday,
    }

    //Create a regex pattern for finding submissions matching identity pubkey
    regex, err := regexp.Compile(strings.Join([]string{".*-", identity.PublicKey, ".json"}, ""))
    if err != nil {
        log.Fatalf("Error creating regular expression out of key: %v\n", err)
    }

    paginatorToday := s3.NewListObjectsV2Paginator(ctx.Client, inputToday)

    var submissionDataToday dg.MetaToBeSaved
    var lastSubmissionTimeString string
    var lastSubmissionTime time.Time
    var uptimeToday []bool
    var uptimeYesterday []bool

    for paginatorToday.HasMorePages() {
        page, err := paginatorToday.NextPage(ctx.Context)
        if err != nil {
            log.Fatalf("Getting next page of paginatorToday (BPU bucket): %v\n", err)
        }

        for _, obj := range page.Contents {

            submissionTime, err := GetSubmissionTime(*obj.Key)
            if err != nil {
                log.Fatalf("Error parsing time: %v\n", err)
            }
            //Open json file only if the pubkey matches the pubkey in the name
            if regex.MatchString(*obj.Key) {
                if (submissionTime.After(config.Period.Start)) && (submissionTime.Before(config.Period.End)) {

                    objHandle, err := ctx.Client.GetObject(ctx.Context, &s3.GetObjectInput{
                        Bucket: ctx.BucketName,
                        Key:    obj.Key,
                    })

                    if err != nil {
                        log.Fatalf("Error getting object from bucket: %v\n", err)
                    }

                    defer objHandle.Body.Close()

                    objContents, err := io.ReadAll(objHandle.Body)
                    if err != nil {
                        log.Fatalf("Error getting creating reader for json: %v\n", err)
                    }

                    err = json.Unmarshal(objContents, &submissionDataToday)
                    if err != nil {
                        log.Fatalf("Error unmarshaling bucket content: %v\n", err)
                    }

                    var remoteAddr string
                    if config.IgnoreIPs {
                        remoteAddr = ""
                    } else {
                        remoteAddr = submissionDataToday.RemoteAddr
                    }

                    if (!config.IgnoreIPs || submissionDataToday.GraphqlControlPort != 0) {
                        if (identity.PublicKey == submissionDataToday.Submitter.String()) && (identity.PublicIp == remoteAddr) && (*identity.graphQLPort == strconv.Itoa(submissionDataToday.GraphqlControlPort)) {

                            currentSubmissionTime, err := time.Parse(time.RFC3339, submissionDataToday.CreatedAt)
                            if err != nil {
                                log.Fatalf("Error parsing time: %v\n", err)
                            }

                            if lastSubmissionTimeString != "" {
                                lastSubmissionTime, err = time.Parse(time.RFC3339, lastSubmissionTimeString)
                                if err != nil {
                                    log.Fatalf("Error parsing time: %v\n", err)
                                }
                            } else {
                                uptimeToday = append(uptimeToday, true)
                                lastSubmissionTimeString = submissionDataToday.CreatedAt
                                continue
                            }

                            if (lastSubmissionTimeString != "") && (currentSubmissionTime.After(lastSubmissionTime.Add(time.Duration(syncPeriod-5) * time.Minute))) {
                                uptimeToday = append(uptimeToday, true)
                                lastSubmissionTimeString = submissionDataToday.CreatedAt
                                continue
                            } else {
                                continue
                            }
                        }
                    } else {
                        if (identity.PublicKey == submissionDataToday.Submitter.String()) && (identity.PublicIp == remoteAddr) {

                            currentSubmissionTime, err := time.Parse(time.RFC3339, submissionDataToday.CreatedAt)
                            if err != nil {
                                log.Fatalf("Error parsing time: %v\n", err)
                            }
                            if lastSubmissionTimeString != "" {
                                lastSubmissionTime, err = time.Parse(time.RFC3339, lastSubmissionTimeString)
                                if err != nil {
                                    log.Fatalf("Error parsing time: %v\n", err)
                                }
                            } else {
                                uptimeToday = append(uptimeToday, true)
                                lastSubmissionTimeString = submissionDataToday.CreatedAt
                                continue
                            }

                            if (lastSubmissionTimeString != "") && (currentSubmissionTime.After(lastSubmissionTime.Add(time.Duration(syncPeriod-5) * time.Minute))) {
                                uptimeToday = append(uptimeToday, true)
                                lastSubmissionTimeString = submissionDataToday.CreatedAt
                                continue
                            } else {
                                continue
                            }
                        }
                    }
                }
            }
        }
    }

    uptimePercent := (float64(len(uptimeToday)+len(uptimeYesterday)) / float64(numberOfSubmissionsNeeded)) * 100

    if uptimePercent > 100.00 {
        uptimePercent = 100.00
    }
    uptimePercentString := strconv.FormatFloat(uptimePercent, 'f', 2, 64)
    *identity.Uptime = uptimePercentString
}
