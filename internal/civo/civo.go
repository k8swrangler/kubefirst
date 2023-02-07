package civo

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/civo/civogo"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

// Some systems fail to resolve TXT records, so try to use Google as a backup
var backupResolver = &net.Resolver{
	PreferGo: true,
	Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
		d := net.Dialer{
			Timeout: time.Millisecond * time.Duration(10000),
		}
		return d.DialContext(ctx, network, "8.8.8.8:53")
	},
}

func TestDomainLiveness(dryRun bool, domainName, domainId, region string) bool {
	if dryRun {
		log.Info().Msg("[#99] Dry-run mode, TestHostedZoneLiveness skipped.")
		return true
	}

	civoRecordName := fmt.Sprintf("kubefirst-liveness.%s", domainName)
	civoRecordValue := "domain record propagated"

	civoClient, err := civogo.NewClient(os.Getenv("CIVO_TOKEN"), region)
	if err != nil {
		log.Info().Msg(err.Error())
		return log.Logger.Fatal().Stack().Enabled()
	}

	log.Info().Msgf("checking to see if record %s exists", domainName)
	log.Info().Msgf("hostedZoneId %s", domainId)
	log.Info().Msgf("route53RecordName %s", domainName)

	civoRecordConfig := &civogo.DNSRecordConfig{
		Type:     civogo.DNSRecordTypeTXT,
		Name:     civoRecordName,
		Value:    civoRecordValue,
		Priority: 100,
		TTL:      10,
	}
	record, err := civoClient.CreateDNSRecord(domainName, civoRecordConfig)
	if err != nil {
		log.Warn().Msgf("%s", err)
		return false
	}

	count := 0
	// todo need to exit after n number of minutes and tell them to check ns records
	// todo this logic sucks
	for count <= 100 {
		count++

		log.Info().Msgf("%s", record.Name)
		ips, err := net.LookupTXT(record.Name)
		if err != nil {
			ips, err = backupResolver.LookupTXT(context.Background(), record.Name)
		}

		log.Info().Msgf("%s", ips)

		if err != nil {
			log.Warn().Msgf("Could not get record name %s - waiting 10 seconds and trying again: \nerror: %s", record.Name, err)
			time.Sleep(10 * time.Second)
		} else {
			for _, ip := range ips {
				// todo check ip against route53RecordValue in some capacity so we can pivot the value for testing
				log.Info().Msgf("%s. in TXT record value: %s\n", record.Name, ip)
				count = 101
			}
		}
		if count == 100 {
			log.Panic().Msg("unable to resolve hosted zone dns record. please check your domain registrar")
		}
	}
	return true
}

// GetDNSInfo try to reach the provided hosted zone
func GetDNSInfo(domainName, region string) (string, error) {

	log.Info().Msg("GetDNSInfo (working...)")

	civoClient, err := civogo.NewClient(os.Getenv("CIVO_TOKEN"), region)
	if err != nil {
		log.Info().Msg(err.Error())
		return "", err
	}

	civoDNSDomain, err := civoClient.FindDNSDomain(domainName)
	if err != nil {
		log.Info().Msg(err.Error())
		return "", err
	}

	//! tracker 3
	// todo: this doesn't default to testing the dns check
	skipHostedZoneCheck := viper.GetBool("init.domain.enabled")
	if !skipHostedZoneCheck {
		hostedZoneLiveness := TestDomainLiveness(globalFlags.DryRun, awsFlags.HostedZoneName, hostedZoneId)
		if !hostedZoneLiveness {
			msg := "failed to check the liveness of the HostedZone. A valid public HostedZone on the same AWS " +
				"account as the one where Kubefirst will be installed is required for this operation to " +
				"complete.\nTroubleshoot Steps:\n\n - Make sure you are using the correct AWS account and " +
				"region.\n - Verify that you have the necessary permissions to access the hosted zone.\n - Check " +
				"that the hosted zone is correctly configured and is a public hosted zone\n - Check if the " +
				"hosted zone exists and has the correct name and domain.\n - If you don't have a HostedZone," +
				"please follow these instructions to create one: " +
				"https://docs.aws.amazon.com/Route53/latest/DeveloperGuide/hosted-zones-working-with.html \n\n" +
				"if you are still facing issues please reach out to support team for further assistance"
			log.Error().Msg(msg)
			return errors.New(msg)
		}
	} else {
		log.Info().Msg("skipping hosted zone check")
	}

	return civoDNSDomain.AccountID, nil

}
