package main

import (
	"C"
	"fmt"
	"log"
	"strconv"
	"time"
	"unsafe"

	"github.com/fluent/fluent-bit-go/output"
	ocilog "github.com/flynnkc/go-oci-log-writer"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/common/auth"
)

var (
	writer  *ocilog.LogWriter
	logType string = "OCI Fluent-Bit Plugin" // Identify logs as coming from plugin
)

func main() {}

func FLBPluginRegister(def unsafe.Pointer) int {
	return output.FLBPluginRegister(def, "oci-logging",
		"Plugin for OCI Logging Service")
}

// FLBPluginInit is where we create instances stored in globals for setup
func FLBPluginInit(plugin unsafe.Pointer) int {
	// Mandatory variables
	logId := output.FLBPluginConfigKey(plugin, "log_id")
	src := output.FLBPluginConfigKey(plugin, "log_source")
	principal := output.FLBPluginConfigKey(plugin, "principal")

	// Get provider defaulting to local config file with default profile
	var p common.ConfigurationProvider
	var err error
	switch principal {
	case "workload":
		p, err = auth.OkeWorkloadIdentityConfigurationProvider()
		if err != nil {
			log.Printf("unable to initialize workload identity provider: %s", err)
			return output.FLB_ERROR
		}
	case "instance":
		p, err = auth.InstancePrincipalConfigurationProvider()
		if err != nil {
			log.Printf("unable to initialize instance identity provider: %s", err)
			return output.FLB_ERROR
		}
	default:
		p = common.DefaultConfigProvider()
	}

	d := ocilog.LogWriterDetails{
		Provider: p,
		LogId:    &logId,
		Type:     &logType,
		Source:   &src,
	}

	// Optional variables
	if subject := output.FLBPluginConfigKey(plugin, "subject"); subject != "" {
		d.Subject = &subject
	}
	if buf := output.FLBPluginConfigKey(plugin, "buffer_size"); buf != "" {
		bufSize, err := strconv.Atoi(buf)
		if err == nil {
			d.BufferSize = &bufSize
		} else {
			log.Printf("invalid buffer size %s, using default", buf)
		}
	}

	writer, err = ocilog.New(d)
	if err != nil {
		log.Printf("unable to initialize workload identity provider: %s", err)
		return output.FLB_ERROR
	}

	return output.FLB_OK
}

func FLBPluginFlushCtx(ctx, data unsafe.Pointer, length C.int, tag *C.char) int {
	dec := output.NewDecoder(data, int(length))

	for {
		ret, ts, record := output.GetRecord(dec)
		if ret != 0 {
			break
		}

		var timestamp time.Time
		switch t := ts.(type) {
		case output.FLBTime:
			timestamp = ts.(output.FLBTime).Time
		case uint64:
			timestamp = time.Unix(int64(t), 0)
		default:
			log.Println("time provided invalid, defaulting to now.")
			timestamp = time.Now()
		}

		str := fmt.Sprintf("%s %s\n", C.GoString(tag), timestamp.String())

		for k, v := range record {
			str += fmt.Sprintf("%s: %s\n", k, v)
		}

		_, err := writer.Write([]byte(str))
		if err != nil {
			return output.FLB_ERROR
		}
	}

	return output.FLB_OK
}

func FLBPluginExit() int {
	writer.Close()

	return output.FLB_OK
}
