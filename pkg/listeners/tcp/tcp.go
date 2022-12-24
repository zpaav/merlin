// Merlin is a post-exploitation command and control framework.
// This file is part of Merlin.
// Copyright (C) 2022  Russel Van Tuyl

// Merlin is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// any later version.

// Merlin is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with Merlin.  If not, see <http://www.gnu.org/licenses/>.

// Package tcp contains the structures and interface for peer-to-peer communications through a TCP bind listener used for Agent communications
// TCP listener's do not have a server because the Merlin Server does not send/receive messages. They are sent through
// peer-to-peer communications
package tcp

import (
	// Standard
	"crypto/sha256"
	"fmt"

	"net"
	"strconv"
	"strings"

	// 3rd Party
	uuid "github.com/satori/go.uuid"

	// Merlin
	"github.com/Ne0nd0g/merlin/pkg/authenticators"
	"github.com/Ne0nd0g/merlin/pkg/authenticators/none"
	"github.com/Ne0nd0g/merlin/pkg/authenticators/opaque"
	"github.com/Ne0nd0g/merlin/pkg/core"
	"github.com/Ne0nd0g/merlin/pkg/listeners"
	"github.com/Ne0nd0g/merlin/pkg/logging"
	"github.com/Ne0nd0g/merlin/pkg/messages"
	"github.com/Ne0nd0g/merlin/pkg/servers"
	"github.com/Ne0nd0g/merlin/pkg/services/agent"
	"github.com/Ne0nd0g/merlin/pkg/transformer"
	"github.com/Ne0nd0g/merlin/pkg/transformer/encoders/gob"
	"github.com/Ne0nd0g/merlin/pkg/transformer/encrypters/aes"
)

// Listener is an aggregate structure that implements the Listener interface
type Listener struct {
	id           uuid.UUID                    // id is the Listener's unique identifier
	auth         authenticators.Authenticator // auth is the process or method to authenticate Agents
	transformers []transformer.Transformer    // transformers is a list of transformers to encode and encrypt Agent messages
	description  string                       // description of the listener
	name         string                       // name of the listener
	options      map[string]string            // options is a map of the listener's configurable options used with NewTCPListener function
	psk          []byte                       // psk is the Listener's Pre-Shared Key used for initial message encryption until the Agent is authenticated
	iface        string                       // iface is the interface generated tcp-bind Agents will listen on; used when compiling TCP Agents
	port         int                          // port is the generated tcp-bind agent will listen on; used when compiling TCP Agents
	agentService *agent.Service               // agentService is used to interact with Agents
}

// NewTCPListener is a factory that creates and returns a Listener aggregate that implements the Listener interface
func NewTCPListener(options map[string]string) (listener Listener, err error) {
	// Create and set the listener's ID
	listener.id = uuid.NewV4()

	// Ensure a listener name was provided
	listener.name = options["Name"]
	if listener.name == "" {
		return listener, fmt.Errorf("a listener name must be provided")
	}

	// Set the description
	if _, ok := options["Description"]; ok {
		listener.description = options["Description"]
	}

	// Set the PSK
	if _, ok := options["PSK"]; ok {
		psk := sha256.Sum256([]byte(options["PSK"]))
		listener.psk = psk[:]
	}

	// Set the Interface
	if options["Interface"] == "" {
		err = fmt.Errorf("a network interface address must be provided")
		return
	}
	ip := net.ParseIP(options["Interface"])
	if ip == nil {
		err = fmt.Errorf("%s is not a valid network interface", options["Interface"])
		return
	}
	listener.iface = options["Interface"]

	// Set the port
	if options["Port"] == "" {
		err = fmt.Errorf("a network interface port must be provided")
		return
	}
	listener.port, err = strconv.Atoi(options["Port"])
	if err != nil {
		err = fmt.Errorf("there was an error converting the port number to an integer: %s", err.Error())
		return
	}

	// Set the Transforms
	if _, ok := options["Transforms"]; ok {
		transforms := strings.Split(options["Transforms"], ",")
		for _, transform := range transforms {
			var t transformer.Transformer
			switch strings.ToLower(transform) {
			case "gob-base":
				t = gob.NewEncoder(gob.BASE)
				//t, err = encoders.New(encoders.GOB, 1)
			case "gob-delegate":
				// TODO I think I can remove this because the linked agent does the GOB decoding of this type
				//t, err = encoders.New(encoders.GOB, 2)
			case "aes":
				t = aes.NewEncrypter()
				//t, err = encrypters.NewEncrypter(encrypters.AES)
			default:
				err = fmt.Errorf("pkg/listeners/tcp.NewTCPListener(): unhandled transform type: %s", transform)
			}
			if err != nil {
				return
			}
			listener.transformers = append(listener.transformers, t)
		}
	}

	// Add the (optional) authenticator
	if _, ok := options["Authenticator"]; ok {
		switch strings.ToLower(options["Authenticator"]) {
		case "opaque":
			listener.auth, err = opaque.NewAuthenticator()
			if err != nil {
				return listener, fmt.Errorf("pkg/listeners/tcp.NewTCPListener(): there was an error getting the authenticator: %s", err)
			}
		default:
			listener.auth = none.NewAuthenticator()
		}
	}

	// Store the passed in options for later
	listener.options = options

	// Add the agent service
	listener.agentService = agent.NewAgentService()

	return listener, nil
}

// DefaultOptions returns a map of configurable listener options that will subsequently be passed to the NewTCPListener function
func DefaultOptions() map[string]string {
	options := make(map[string]string)
	options["Name"] = "My TCP Listener"
	options["Description"] = "Default TCP Listener"
	options["Interface"] = "127.0.0.1"
	options["Port"] = "7777"
	options["PSK"] = "merlin"
	options["Transforms"] = "aes,gob-base"
	options["Protocol"] = "TCP"
	options["Authenticator"] = "OPAQUE"
	return options
}

// Authenticate takes data coming into the listener from an agent and passes it to the listener's configured
// authenticator to authenticate the agent. Once an agent is authenticated, this function will no longer be used.
func (l *Listener) Authenticate(id uuid.UUID, data interface{}) (messages.Base, error) {
	auth := l.auth
	return auth.Authenticate(id, data)
}

// ConfiguredOptions returns the server's current configuration for options that can be set by the user
func (l *Listener) ConfiguredOptions() (options map[string]string) {
	options = make(map[string]string)
	options["Description"] = l.description
	options["ID"] = l.id.String()
	options["Name"] = l.name
	options["PSK"] = l.PSK()
	options["Transforms"] = ""
	for _, transform := range l.transformers {
		options["Transforms"] += transform.String()
	}
	return options
}

// Construct takes in a messages.Base structure that is ready to be sent to an agent and runs all the data transforms
// on it to encode and encrypt it. If an empty key is passed in, then the listener's interface encryption key will be used.
func (l *Listener) Construct(msg messages.Base, key []byte) (data []byte, err error) {
	if core.Debug {
		logging.Message("debug", fmt.Sprintf("pkg/listeners.Construct(): entering into function with Base message: %+v and key: %x", msg, key))
	}

	//fmt.Printf("pkg/listeners.Construct(): entering into function with Base message: %+v and key: %x\n", msg, key)
	// TODO Message padding

	if len(key) == 0 {
		key = l.psk
	}

	for i := len(l.transformers); i > 0; i-- {

		if i == len(l.transformers) {
			//fmt.Printf("TCP construct transformer %T: %+v\n", l.transformers[i-1], l.transformers[i-1])
			// First call should always take a Base message
			data, err = l.transformers[i-1].Construct(msg, key)
		} else {
			//fmt.Printf("TCP construct transformer %T: %+v\n", l.transformers[i-1], l.transformers[i-1])
			data, err = l.transformers[i-1].Construct(data, key)
		}
		if err != nil {
			return nil, fmt.Errorf("pkg/listeners.Construct(): there was an error calling the transformer construct function: %s", err)
		}
	}
	// Prepend agent UUID bytes so outside functions can determine the ID
	//data = append(msg.ID.Bytes(), data...)
	//fmt.Printf("Returning data(%d) and error: %v\n", len(data), err)
	return
}

// Deconstruct takes in data that an agent sent to the listener and runs all the listener's transforms on it until
// a messages.Base structure is returned. The key is used for decryption transforms. If an empty key is passed in, then
// the listener's interface encryption key will be used.
func (l *Listener) Deconstruct(data, key []byte) (messages.Base, error) {
	if core.Debug {
		logging.Message("debug", fmt.Sprintf("pkg/listeners/tcp.Deconstruct(): entering into function with Data length %d and key: %x", len(data), key))
	}
	//fmt.Printf("pkg/listeners/tcp.Deconstruct(): entering into function with Data length %d and key: %x\n", len(data), key)

	// Get the listener's interface encryption key
	if len(key) == 0 {
		key = l.psk
	}

	for _, transform := range l.transformers {
		//fmt.Printf("TCP deconstruct transformer %T: %+v\n", transform, transform)
		ret, err := transform.Deconstruct(data, key)
		if err != nil {
			return messages.Base{}, err
		}
		switch ret.(type) {
		case []uint8:
			data = ret.([]byte)
		case string:
			data = []byte(ret.(string)) // Probably not what I should be doing
		case messages.Base:
			//fmt.Printf("pkg/listeners/tcp.Deconstruct(): returning Base message: %+v\n", ret.(messages.Base))
			return ret.(messages.Base), nil
		default:
			return messages.Base{}, fmt.Errorf("pkg/listeners.Deconstruct(): unhandled data type for Deconstruct(): %T", ret)
		}
	}
	return messages.Base{}, fmt.Errorf("pkg/listeners/tcp.Deconstruct(): unable to transform data into messages.Base structure")

}

// Description returns the listener's description
func (l *Listener) Description() string {
	return l.description
}

// ID returns the listener's unique identifier
func (l *Listener) ID() uuid.UUID {
	return l.id
}

// Name returns the listener's name
func (l *Listener) Name() string {
	return l.name
}

// Options returns the original map of options passed into the NewTCPListener function
func (l *Listener) Options() map[string]string {
	return l.options
}

// Protocol returns a constant from the listeners package that represents the protocol type of this listener
func (l *Listener) Protocol() int {
	return listeners.TCP
}

// PSK returns the listener's pre-shared key used for encrypting & decrypting agent messages
func (l *Listener) PSK() string {
	return string(l.psk)
}

// Server is not used by TCP listeners because the Merlin Server itself does not listen for or send Agent messages.
// TCP listeners are used for peer-to-peer communications that come in from other listeners like the HTTP listener.
// This functions returns nil because it is not used but required to implement the interface.
func (l *Listener) Server() *servers.ServerInterface {
	return nil
}

// String returns the listener's name
func (l *Listener) String() string {
	return l.name
}

// SetOption sets the value for a configurable option on the Listener
func (l *Listener) SetOption(option string, value string) error {
	switch strings.ToLower(option) {
	case "name":
		l.name = value
	case "description":
		l.description = value
	default:
		return fmt.Errorf("pkg/listeners/tcp.SetOptions(): unhandled option %s", option)
	}
	return nil
}

// Status returns the status of the embedded server's state, required to implement the Listener interface.
// TCP Listeners do not have an embedded server and therefore returns a static "Created"
func (l *Listener) Status() string {
	return "Created"
}
