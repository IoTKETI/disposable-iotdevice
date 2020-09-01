// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2018-2020 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

// This package provides a implementation of a ProtocolDriver interface.
//
package driver

import (
	"fmt"
	"sync"
	"time"
	"net/http"
	"bytes"
	"encoding/json"
	"io/ioutil"
	"io"
	"strconv"

	dsModels "github.com/edgexfoundry/device-sdk-go/pkg/models"
	"github.com/edgexfoundry/go-mod-core-contracts/clients/logger"
	"github.com/edgexfoundry/go-mod-core-contracts/models"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"gw/ObjectTypeParameters"
	"gw/ResourceName"
	"gw/Parameters"
	DeviceRegistration "gw/1.DeviceRegistration"
	DeviceMicroserviceInformationReport "gw/2.DeviceMicroserviceInformationReport"
	MicroserviceCreation "gw/10.MicroserviceCreation"
	MicroserviceRun "gw/11.MicroserviceRun"
	MicroserviceStop "gw/15.MicroserviceStop"
	TaskParameterSet "gw/21.TaskParameterSet"
	MicroserviceOutputReport "gw/14.MicroserviceOutputReport"
)

var once sync.Once
var driver *DisposableDriver
var mqttClient mqtt.Client
var mis map[string]interface{} = make(map[string]interface{})
var global_dri int = 100

type DisposableDriver struct {
	lc            logger.LoggingClient
	asyncCh       chan<- *dsModels.AsyncValues
}

func NewProtocolDriver() dsModels.ProtocolDriver {
	once.Do(func() {
		driver = new(DisposableDriver)
	})
	return driver
}

func MsgHandler(client mqtt.Client, mqtt_msg mqtt.Message) {
        data := mqtt_msg.Payload()
        params := Parameters.NewParameter()

        fmt.Println(string(data))
        if bytes.Contains(data,[]byte("if")){
                ParseRequestMsg(data,params)
		global_dri++
        } else {
                ParseResponseMsg(data,params)
        }

}

func SendDownlinkMsg(msg string){
        mqttClient.Publish("ble_downlink",0,false,msg)
}

func (d *DisposableDriver) DisconnectDevice(deviceName string, protocols map[string]models.ProtocolProperties) error {
	d.lc.Info(fmt.Sprintf("DisposableDriver.DisconnectDevice: disposable-device driver is disconnecting to %s", deviceName))
	return nil
}

func (d *DisposableDriver) Initialize(lc logger.LoggingClient, asyncCh chan<- *dsModels.AsyncValues, deviceCh chan<- []dsModels.DiscoveredDevice) error {
	d.lc = lc
	d.asyncCh = asyncCh

	opts := mqtt.NewClientOptions().AddBroker("localhost:1883")
	var MsgHandler mqtt.MessageHandler = MsgHandler

	opts.SetDefaultPublishHandler(MsgHandler)
	mqttClient = mqtt.NewClient(opts)
	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	if token := mqttClient.Subscribe("ble_uplink",0,nil); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		return nil
	}

	return nil
}

func (d *DisposableDriver) HandleReadCommands(deviceName string, protocols map[string]models.ProtocolProperties, reqs []dsModels.CommandRequest) (res []*dsModels.CommandValue, err error) {
	for _, req := range reqs {
		for k, v := range req.Attributes {
			fmt.Println(req.DeviceResourceName + ", " + k + " : " + v)
		}
	}
	now := time.Now().UnixNano()
	res = make([]*dsModels.CommandValue, len(reqs))
	for i, req := range reqs {
		switch req.DeviceResourceName {
		case "mis":
			res[i],_ = dsModels.NewUint32ArrayValue(req.DeviceResourceName, now, mis[deviceName].([]uint32))
		}
	}

	return res, nil
}

func (d *DisposableDriver) HandleWriteCommands(deviceName string, protocols map[string]models.ProtocolProperties, reqs []dsModels.CommandRequest,
params []*dsModels.CommandValue) error {

	switch(len(reqs)){
	case 1:
		for _, param := range params {
			switch param.DeviceResourceName {
			case "mis":
				vArr, err := param.Uint32ArrayValue()
				if err != nil {
					return fmt.Errorf("DisposableDriver.HandleWriteCommands: %v", err)
				}

				fmt.Printf("vArr: %d\n", vArr)
				if mis[deviceName] == nil{
					mis[deviceName] = make([]uint32, len(vArr))
				}
				for i, v := range vArr{
					mis[deviceName].([]uint32)[i] = v
				}
				//mqttClient.Publish("edgex", 0, false, "ID:"+deviceName+",cmd:"+v)
			case "mistocreate":
				vArr, err := param.Uint32ArrayValue()
				if err != nil {
					return fmt.Errorf("DisposableDriver.HandleWriteCommands: %v", err)
				}

				fmt.Printf("vArr: %d\n", vArr)
				mis := make([]int, len(vArr))

				for k,v := range(vArr){
					mis[k] = int(v)
				}

				parameters := Parameters.NewParameter()
				parameters.SetInterfaceID(10)
				parameters.SetDisposableIoTRequestID(global_dri)
				global_dri++
				parameters.SetMicroserviceIDs(mis)
				MicroserviceCreation.Request(parameters,mqttClient)

				break;
			case "mistorun":
				vArr, err := param.Uint32ArrayValue()
				if err != nil {
					return fmt.Errorf("DisposableDriver.HandleWriteCommands: %v", err)
				}

				fmt.Printf("vArr: %d\n", vArr)
				mis := make([]int, len(vArr))

				for k,v := range(vArr){
					mis[k] = int(v)
				}

				parameters := Parameters.NewParameter()
				parameters.SetInterfaceID(11)
				parameters.SetDisposableIoTRequestID(global_dri)
				global_dri++
				parameters.SetMicroserviceIDs(mis)
				MicroserviceRun.Request(parameters, mqttClient)

				break;
			case "mistostop":
				vArr, err := param.Uint32ArrayValue()
				if err != nil {
					return fmt.Errorf("DisposableDriver.HandleWriteCommands: %v", err)
				}

				fmt.Printf("vArr: %d\n", vArr)

				mis := make([]int, len(vArr))

				for k,v := range(vArr){
					mis[k] = int(v)
				}

				parameters := Parameters.NewParameter()
				parameters.SetInterfaceID(15)
				parameters.SetDisposableIoTRequestID(global_dri)
				global_dri++
				parameters.SetMicroserviceIDs(mis)
				MicroserviceStop.Request(parameters, mqttClient)
				break;
			default:
				return fmt.Errorf("DisposableDriver.HandleWriteCommands: there is no matched device resource for %s", param.String())
			}
		}
		break;
	case 4:
		parameters := Parameters.NewParameter()
		fp := ObjectTypeParameters.Fp{}
		for _, param := range params {
			switch param.DeviceResourceName {
			case "mitoset":
				mi,_ := param.StringValue()
				mis := make([]int,1)
				v,_ := strconv.Atoi(mi)
				mis[0] = v
				parameters.SetMicroserviceIDs(mis)
				break;
			case "titoset":
				ti,_ := param.StringValue()
				tis := make([]int,1)
				v,_ := strconv.Atoi(ti)
				tis[0] = v
				parameters.SetTaskIDs(tis)
				break;
			case "oprd":
				oprd,err := param.StringValue()
				if err == nil {
					fmt.Println("OKOK")
					fp.Oprd = string(oprd)
				}
				break;
			case "sprd":
				sprd,err := param.StringValue()
				if err == nil {
					fp.Sprd = string(sprd)
				}
				break;
			}
		}
		parameters.SetInterfaceID(21)
		parameters.SetDisposableIoTRequestID(global_dri)
		global_dri++
		/*parameters.SetMicroserviceIDs([]int{1})
		parameters.SetTaskIDs([]int{1})
		fp := ObjectTypeParameters.Fp{}
		fp.Oprd ="C"*/
		parameters.SetFlexibleTaskParameter(fp)
		TaskParameterSet.Request(parameters,mqttClient)
		break;
	}

	return nil
}

func (d *DisposableDriver) Stop(force bool) error {
	d.lc.Info("DisposableDriver.Stop: device-random driver is stopping...")
	return nil
}

func (d *DisposableDriver) AddDevice(deviceName string, protocols map[string]models.ProtocolProperties, adminState models.AdminState) error {
	d.lc.Debug(fmt.Sprintf("a new Device is added: %s", deviceName))
	return nil
}

func (d *DisposableDriver) UpdateDevice(deviceName string, protocols map[string]models.ProtocolProperties, adminState models.AdminState) error {
	d.lc.Debug(fmt.Sprintf("Device %s is updated", deviceName))
	return nil
}

func (d *DisposableDriver) RemoveDevice(deviceName string, protocols map[string]models.ProtocolProperties) error {
	d.lc.Debug(fmt.Sprintf("Device %s is removed", deviceName))
	return nil
}

func ParseRequestMsg(data []byte, parameters *Parameters.Parameter) {

	headerTemp := make(map[string]string)
	driIf := make(map[int]string)
	temp := bytes.SplitAfterN(data, []byte("}"), 2)  //헤더 , 바디 분리 , n은 2개로 "}" 나뉜다
	fmt.Println(string(temp[0]))
	header := bytes.SplitAfter(bytes.ReplaceAll(bytes.ReplaceAll(temp[0], []byte("{"), []byte("")), []byte("}"), []byte("")), []byte(";"))
	fmt.Println(string(header[0]))
	for j := range header {
		headerTemp[string(bytes.Split(bytes.ReplaceAll(bytes.TrimSpace(header[j]), []byte(";"), []byte("")), []byte("="))[0])] = string(bytes.Split(bytes.ReplaceAll(bytes.TrimSpace(header[j]), []byte(";"), []byte("")), []byte("="))[1])
	}
	for k,v := range headerTemp{
		fmt.Println(k+","+v)
	}
	fmt.Println(headerTemp["if"])

		switch headerTemp["if"] {

		case ResourceName.DeviceRegistration:

			DeviceRegistration.RQparsing(data,parameters)
			parameters.SetResponseStatusCode(200)
			res := DeviceRegistration.Response(parameters)
			SendDownlinkMsg(res)

			driIf[parameters.DisposableIoTRequestID()] = ResourceName.DeviceRegistration
			break
		case ResourceName.DeviceMicroserviceInformationReport:

			DeviceMicroserviceInformationReport.RQparsing(data, parameters)
//			deviceInfo[parameters.DeviceID()] = parameters
			parameters.SetResponseStatusCode(200)
			res := DeviceMicroserviceInformationReport.Response(parameters)
			SendDownlinkMsg(res)
			driIf[parameters.DisposableIoTRequestID()] = ResourceName.DeviceMicroserviceInformationReport
			break
		case ResourceName.DeviceTaskInformationRequest:


			driIf[parameters.DisposableIoTRequestID()] = ResourceName.DeviceTaskInformationRequest
			break
		case ResourceName.TaskRun:

			driIf[parameters.DisposableIoTRequestID()] = ResourceName.TaskRun
			break
		case ResourceName.MicroserviceOutputReport:
			MicroserviceOutputReport.RQparsing(data,parameters)
			fmt.Println(parameters.OutputParameter())

			resp, err := http.Get("http://localhost:48081/api/v1/device/name/"+parameters.DeviceID())
	                defer resp.Body.Close()

		        dev_info_json, _ := ioutil.ReadAll(resp.Body)
			dev_info := make(map[string]string)
	                json.Unmarshal(dev_info_json, &dev_info)
	                edgex_id := dev_info["id"]

			fmt.Println(edgex_id)

			body := bytes.NewBufferString("{\"device\":\""+edgex_id+"\",\"readings\":[{\"name\":\"atemp\", \"value\":\""+parameters.OutputParameter()+"\"}]}")
			fmt.Println(body)
			resp, err = http.Post("http://localhost:48080/api/v1/event", "text/plain", body)
			fmt.Println("OK")
			fmt.Println(body)
	                if err != nil {
		                panic(err)
			}
	                if resp != nil {
	                        defer resp.Body.Close()
	                }

		        io.Copy(ioutil.Discard, resp.Body)

			parameters.SetResponseStatusCode(200)
			res := MicroserviceOutputReport.Response(parameters,mqttClient)
			SendDownlinkMsg(res)
			//parameters.DriIf()[parameters.DisposableIoTRequestID()] = ResourceName.MicroserviceOutputReport

			driIf[parameters.DisposableIoTRequestID()] = ResourceName.MicroserviceOutputReport
			break
		case ResourceName.TaskOff:

			driIf[parameters.DisposableIoTRequestID()] = ResourceName.TaskOff
			break
		}


	parameters.SetDriIf(driIf)

}

func ParseResponseMsg(data []byte,parameters *Parameters.Parameter)  {


	switch parameters.DriIf()[parameters.DisposableIoTRequestID()] {

	case ResourceName.DeviceRegistration:

		DeviceRegistration.RSparsing(data,parameters)

		delete(parameters.DriIf(),parameters.DisposableIoTRequestID())
		break
	case ResourceName.DeviceMicroserviceInformationReport:



		delete(parameters.DriIf(),parameters.DisposableIoTRequestID())
		break
	case ResourceName.DeviceTaskInformationRequest:

		delete(parameters.DriIf(),parameters.DisposableIoTRequestID())
		break
	case ResourceName.TaskRun:

		delete(parameters.DriIf(),parameters.DisposableIoTRequestID())
		break
	case ResourceName.MicroserviceOutputReport:

		delete(parameters.DriIf(),parameters.DisposableIoTRequestID())
		break
	case ResourceName.TaskOff:

		delete(parameters.DriIf(),parameters.DisposableIoTRequestID())
		break
	}
}

