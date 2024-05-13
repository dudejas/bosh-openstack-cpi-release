package services

import (
	"context"
	"errors"
	"fmt"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/utils"
	"github.com/gophercloud/gophercloud"
	"net"
	"time"
)

const maxRetries = 10

var DefaultRetrySleepDuration = 3 * time.Second

func RetryOnError(logger utils.Logger) func(
	ctx context.Context,
	method string, url string,
	options *gophercloud.RequestOpts,
	inError error,
	failCount uint,
) error {
	return func(ctx context.Context, method, url string, options *gophercloud.RequestOpts, inError error, failCount uint) error {
		if failCount >= maxRetries {
			return fmt.Errorf("max retry attempts (%d) reached, err: %w", failCount, inError)
		}

		logger.Warn(
			"retry on error",
			fmt.Sprintf("attempt failed with error: %v", inError))

		sleepDuration := DefaultRetrySleepDuration
		var responseCode gophercloud.ErrUnexpectedResponseCode
		if errors.As(inError, &responseCode) {
			if responseCode.Actual == 500 || responseCode.Actual == 503 {
				logger.Warn(
					"retry on error",
					fmt.Sprintf("detected HTTP error %d, sleeping for %.0f seconds", responseCode.Actual, sleepDuration.Seconds()))
				time.Sleep(sleepDuration)
				return nil
			}
		} else if isTimeout(inError) {
			logger.Warn(
				"retry on error",
				fmt.Sprintf("detected timeout, sleeping for %.0f seconds", sleepDuration.Seconds()))
			time.Sleep(sleepDuration)
			return nil
		} else if isNetworkError(inError) {
			logger.Warn(
				"retry on error",
				fmt.Sprintf("detected network error, sleeping for %.0f seconds", sleepDuration.Seconds()))
			time.Sleep(sleepDuration)
			return nil
		}

		return inError
	}
}

func isTimeout(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout()
	}
	return false
}

func isNetworkError(err error) bool {
	_, ok := err.(*net.OpError)
	if ok {
		return true
	}
	return false
}
