/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package helpers

import (
	"fmt"
	"os"
	"path/filepath"
)

var configTemplate = `version: 1.0.0

client:
  logging:
    level: info
  cryptoconfig:
    path: %s 
  credentialStore:
    path: "%s/state-store"
    cryptoStore:
      path: %s/msp

  BCCSP:
    security:
     enabled: true
     default:
      provider: "SW"
     hashAlgorithm: "SHA2"
     softVerify: true
     level: 256
channels:
  %s:
    peers:
      peer0.org1.example.com:
        endorsingPeer: true
        chaincodeQuery: true
        ledgerQuery: true
        eventSource: true
      peer1.org1.example.com:
        endorsingPeer: true
        chaincodeQuery: true
        ledgerQuery: true
        eventSource: true
      peer0.org2.example.com:
        endorsingPeer: true
        chaincodeQuery: true
        ledgerQuery: true
        eventSource: true
      peer1.org2.example.com:
        endorsingPeer: true
        chaincodeQuery: true
        ledgerQuery: true
        eventSource: true
organizations:
  org1:
    mspid: Org1MSP
    cryptoPath:  peerOrganizations/org1.example.com/users/{username}@org1.example.com/msp
    peers:
      - peer0.org1.example.com
      - peer1.org1.example.com
  org2:
    mspid: Org2MSP
    cryptoPath:  peerOrganizations/org2.example.com/users/{username}@org2.example.com/msp
    peers:
      - peer0.org2.example.com
      - peer1.org2.example.com
  ordererorg:
      mspID: OrdererMSP
      cryptoPath: ordererOrganizations/example.com/users/{username}@example.com/msp
orderers:
  orderer.example.com:
    url: orderer.example.com:%d
    grpcOptions:
      ssl-target-name-override: orderer.example.com
      keep-alive-time: 0s
      keep-alive-timeout: 20s
      keep-alive-permit: false
      fail-fast: false
      allow-insecure: false
    tlsCACerts:
      path: %s/ordererOrganizations/example.com/tlsca/tlsca.example.com-cert.pem
peers:
  peer0.org1.example.com:
    url: peer0.org1.example.com:%d
    grpcOptions:
      ssl-target-name-override: peer0.org1.example.com
      keep-alive-time: 0s
      keep-alive-timeout: 20s
      keep-alive-permit: false
      fail-fast: false
      allow-insecure: false
    tlsCACerts:
      path: %s/peerOrganizations/org1.example.com/tlsca/tlsca.org1.example.com-cert.pem

  peer1.org1.example.com:
    url: peer1.org1.example.com:%d
    grpcOptions:
      ssl-target-name-override: peer1.org1.example.com
      keep-alive-time: 0s
      keep-alive-timeout: 20s
      keep-alive-permit: false
      fail-fast: false
      allow-insecure: false
    tlsCACerts:
      path: %s/peerOrganizations/org1.example.com/tlsca/tlsca.org1.example.com-cert.pem

  peer0.org2.example.com:
    url: peer0.org2.example.com:%d
    grpcOptions:
      ssl-target-name-override: peer0.org2.example.com
      keep-alive-time: 0s
      keep-alive-timeout: 20s
      keep-alive-permit: false
      fail-fast: false
      allow-insecure: false
    tlsCACerts:
      path: %s/peerOrganizations/org2.example.com/tlsca/tlsca.org2.example.com-cert.pem

  peer1.org2.example.com:
    url: peer0.org2.example.com:%d
    grpcOptions:
      ssl-target-name-override: peer1.org2.example.com
      keep-alive-time: 0s
      keep-alive-timeout: 20s
      keep-alive-permit: false
      fail-fast: false
      allow-insecure: false
    tlsCACerts:
      path: %s/peerOrganizations/org2.example.com/tlsca/tlsca.org2.example.com-cert.pem

entityMatchers:
  peer:
    - pattern: peer0.org1.example.(\w+)
      urlSubstitutionExp: localhost:%d
      sslTargetOverrideUrlSubstitutionExp: peer0.org1.example.com
      mappedHost: peer0.org1.example.com

    - pattern: peer1.org1.example.(\w+)
      urlSubstitutionExp: localhost:%d
      sslTargetOverrideUrlSubstitutionExp: peer1.org1.example.com
      mappedHost: peer1.org1.example.com

    - pattern: peer0.org2.example.(\w+)
      urlSubstitutionExp: localhost:%d
      sslTargetOverrideUrlSubstitutionExp: peer0.org2.example.com
      mappedHost: peer0.org2.example.com

    - pattern: peer1.org2.example.(\w+)
      urlSubstitutionExp: localhost:%d
      sslTargetOverrideUrlSubstitutionExp: peer1.org2.example.com
      mappedHost: peer1.org2.example.com

    - pattern: (\w+).org1.example.(\w+):(\d+)
      urlSubstitutionExp: localhost:$2
      sslTargetOverrideUrlSubstitutionExp: $1.org1.example.com
      mappedHost: $1.org1.example.com

    - pattern: (\w+).org2.example.(\w+):(\d+)
      urlSubstitutionExp: localhost:$2
      sslTargetOverrideUrlSubstitutionExp: $1.org2.example.com
      mappedHost: $1.org2.example.com

    - pattern: (\w+):%d
      urlSubstitutionExp: localhost:%d
      sslTargetOverrideUrlSubstitutionExp: peer0.org1.example.com
      mappedHost: peer0.org1.example.com

    - pattern: (\w+):%d
      urlSubstitutionExp: localhost:%d
      sslTargetOverrideUrlSubstitutionExp: peer1.org1.example.com
      mappedHost: peer1.org1.example.com

    - pattern: (\w+):%d
      urlSubstitutionExp: localhost:%d
      sslTargetOverrideUrlSubstitutionExp: peer0.org2.example.com
      mappedHost: peer0.org2.example.com

    - pattern: (\w+):%d
      urlSubstitutionExp: localhost:%d
      sslTargetOverrideUrlSubstitutionExp: peer1.org2.example.com
      mappedHost: peer1.org2.example.com

  orderer:
    - pattern: (\w+):%d 
      urlSubstitutionExp: localhost:%d
      sslTargetOverrideUrlSubstitutionExp: orderer.example.com
      mappedHost: orderer.example.com
    - pattern: (\w+).example.(\w+)
      urlSubstitutionExp: localhost:%d
      sslTargetOverrideUrlSubstitutionExp: orderer.example.com
      mappedHost: orderer.example.com`

func CreateProxyConfig(testDir, channelName, cryptoConfigPath string, org1Peer0Port, org1Peer1Port, org2Peer0Port, org2Peer1Port, ordererPort uint16) (string, error) {
	config := fmt.Sprintf(configTemplate,
		cryptoConfigPath,
		testDir, testDir,
		channelName,
		ordererPort,
		cryptoConfigPath,
		org1Peer0Port, cryptoConfigPath,
		org1Peer1Port, cryptoConfigPath,
		org2Peer0Port, cryptoConfigPath,
		org2Peer1Port, cryptoConfigPath,
		org1Peer0Port,
		org1Peer1Port,
		org2Peer0Port,
		org2Peer1Port,
		org1Peer0Port,
		org1Peer0Port,
		org1Peer1Port,
		org1Peer1Port,
		org2Peer0Port,
		org2Peer0Port,
		org2Peer1Port,
		org2Peer1Port,
		ordererPort,
		ordererPort,
		ordererPort,
	)

	file, err := os.Create(filepath.Join(testDir, "web3-sdk-config.yaml"))
	if err != nil {
		return "", err
	}
	defer file.Close()

	_, err = file.WriteString(config)
	if err != nil {
		return "", err
	}

	return file.Name(), nil
}
