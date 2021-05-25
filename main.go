package main

import (
	"flag"
	"github.com/fatih/color"
	log "github.com/sirupsen/logrus"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

/*CA client cert rotation:
	Take an input from user is Min_cert_duration for which a any cert life can be defined.
	Any Cert duration - expiryWindow should be greater than Min_cert_duration
Approach/Formula for minimum certificate duration:
	CA expiryWindow should be greater than Min_cert_duration
	This values should be satisfied when the node/client certificate
Node/client cron schedule formula:
	client/node cron schedule = Min_cert_duration - delta (very small delta)
*/

var (
	caDurationFlag, nodeDurationFlag, clientDurationFlag string
	caExpiryFlag, nodeExpiryFlag, clientExpiryFlag       string
	minCertDurationFlag                                  string
)

type cert struct {
	Duration     time.Duration
	ValidFrom    time.Time
	ValidTo      time.Time
	LastRotation time.Time
	NexRotation  time.Time
}

func init() {
	flag.StringVar(&caDurationFlag, "ca-duration", "1095d", "duration of ca cert. Defaults to 1095d (3 years)")
	flag.StringVar(&caExpiryFlag, "ca-expiry", "28d", "duration of ca cert. Defaults to 28d")

	flag.StringVar(&nodeDurationFlag, "node-duration", "365d", "duration of ca cert. Defaults to 365h (1 year)")
	flag.StringVar(&nodeExpiryFlag, "node-expiry", "7d", "duration of ca cert. Defaults to 7d")

	flag.StringVar(&clientDurationFlag, "client-duration", "30d", "duration of ca cert. Defaults to 30d")
	flag.StringVar(&clientExpiryFlag, "client-expiry", "2d", "duration of ca cert. Defaults to 2d")

	flag.StringVar(&minCertDurationFlag, "min-cert-duration", "27d", "duration of ca cert. Defaults to 27d")
}

func main() {
	flag.Parse()
	var (
		caDuration, caExpiry, nodeDuration, nodeExpiry, clientDuration, clientExpiry, minCertDuration time.Duration
	)

	// parse all the durations
	caDuration = parseDuration(caDurationFlag)
	caExpiry = parseDuration(caExpiryFlag)
	nodeDuration = parseDuration(nodeDurationFlag)
	nodeExpiry = parseDuration(nodeExpiryFlag)
	clientDuration = parseDuration(clientDurationFlag)
	clientExpiry = parseDuration(clientExpiryFlag)
	minCertDuration = parseDuration(minCertDurationFlag)

	if caDuration.Hours()-caExpiry.Hours() < minCertDuration.Hours() {
		log.Panic("CA cert details do not meet the min-cert-duration criteria")
	}

	if caExpiry.Hours() < minCertDuration.Hours() {
		log.Panic("ca-expiry does not meet the min-cert-duration criteria")
	}

	if nodeDuration.Hours()-nodeExpiry.Hours() <= minCertDuration.Hours() {
		log.Panic("Node cert detauls do not meet the min-cert-duration criteria")
	}

	if clientDuration.Hours()-clientExpiry.Hours() <= minCertDuration.Hours() {
		log.Panic("Client cert details do not meet the min-cert-duration criteria")
	}

	caCert := cert{
		Duration:    caDuration,
		ValidFrom:   time.Now(),
		ValidTo:     time.Now().Add(caDuration),
		NexRotation: time.Now().Add(caDuration - caExpiry),
	}

	nodeCert := cert{
		Duration:  nodeDuration,
		ValidFrom: time.Now(),
		ValidTo:   time.Now().Add(nodeDuration),
	}

	clientCert := cert{
		Duration:  clientDuration,
		ValidFrom: time.Now(),
		ValidTo:   time.Now().Add(clientDuration),
	}

	log.Infof("CA cert duration is                	 [%v] hours", caDuration.Hours())
	log.Infof("CA expirty duration is             	 [%v] hours", caExpiry.Hours())
	log.Infof("Node cert duration is              	 [%v] hours", nodeDuration.Hours())
	log.Infof("Node expiry duration is            	 [%v] hours", nodeExpiry.Hours())
	log.Infof("Client cert duration is            	 [%v] hours", clientDuration.Hours())
	log.Infof("Client expiry duration is          	 [%v] hours", clientExpiry.Hours())
	log.Infof("Min-cert-duration cert duration is    [%v] hours", minCertDuration.Hours())
	log.Infof("curent time is [%v]\n\n", time.Now().Format(time.Stamp))

	log.Print("=============================================================================================")

	delta:= time.Time{}
	delta = delta.Add(1 * time.Hour)

	i := 1
	for t := time.Now().Add(minCertDuration); ; {

		color.Green("Cron: [%d] at [%s] \n\n", i, t.Format(time.Stamp))

		// if client cert expires before next cron, then rotate
		if !clientCert.ValidTo.After(t.Add(minCertDuration)) {
			log.Info("Rotating client cert")
			newDuration := t.Add(clientCert.Duration)

			// if new client cert outlives CA cert life then create client cert valid till CA Duration minus delta
			if newDuration.After(caCert.ValidTo) {

				clientCert.ValidTo = caCert.ValidTo.Add(-1 * time.Hour)
			} else {
				clientCert.ValidTo = newDuration
			}

			clientCert.ValidFrom = t

			log.Infof("Rotated client cert at    [%s]", t.Format(time.Stamp))
			log.Infof("New client cert validTill [%s]\n\n", clientCert.ValidTo.Format(time.Stamp))

		}

		// if node cert expires before next cron, then rotate
		if !nodeCert.ValidTo.After(t.Add(minCertDuration)) {
			log.Info("Rotating node cert")

			newDuration := t.Add(nodeCert.Duration)

			// if new client cert outlives CA cert life then node cert valid till CA Duration minus delta
			if newDuration.After(caCert.ValidTo) {
				nodeCert.ValidTo = caCert.ValidTo.Add(-1 * time.Hour)
			} else {
				nodeCert.ValidTo = newDuration
			}

			nodeCert.ValidFrom = t

			log.Infof("Rotated node cert at      [%s]", t.Format(time.Stamp))
			log.Infof("New node cert validTill   [%s]\n\n", nodeCert.ValidTo.Format(time.Stamp))
		}

		// to imitate 2 cron functionality, check if CA cert rotation if before next client/node cron
		// If Yes rotate CA
		if !caCert.NexRotation.After(t.Add(minCertDuration)) {
			log.Info("Rotating CA cert")
			newDuration := t.Add(caCert.Duration)
			caCert.ValidTo = newDuration
			clientCert.ValidFrom = t
			caCert.NexRotation = t.Add(caDuration - caExpiry)
			log.Infof("Rotated client cert at    [%s]", t.Format(time.Stamp))
			log.Infof("New CA cert validTill     [%s]", caCert.ValidTo.Format(time.Stamp))
			color.Yellow("Next CA cron at           [%s]\n\n", caCert.NexRotation.Format(time.Stamp))

		}

		// next cron schedule
		t = t.Add(minCertDuration)

		log.Print("=============================================================================================")
		i++
		if i == 30 {
			log.Info("Cert rotation will work with these values, verification passed. Exiting")
			os.Exit(1)
		}
	}
}

func parseDuration(duration string) time.Duration {
	var (
		hourDuration time.Duration
		err          error
		day          int
	)

	if hourDuration, err = time.ParseDuration(duration); err == nil {
		return hourDuration
	}

	log.Warningf("failed to parse duration %s", duration)

	// parse day duration as Go only supports "s","m","h"
	r := regexp.MustCompile("^[0-9]*d$")

	if r.Match([]byte(duration)) {
		if day, err = strconv.Atoi(strings.TrimSuffix(duration, "d")); err == nil {
			hourDuration = time.Duration(day) * 24 * time.Hour
			return hourDuration
		}
	}
	log.Panicf("failed to parse duration %s", duration)

	return hourDuration
}
