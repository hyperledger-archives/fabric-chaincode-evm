#!groovy

// Copyright IBM Corp All Rights Reserved
//
// SPDX-License-Identifier: Apache-2.0
//

// Jenkinsfile get triggered when a patchset a submitted or merged
@Library("fabric-ci-lib") _ // global shared library from ci-management repository
// global shared library from ci-management repository
// https://github.com/hyperledger/ci-management/tree/master/vars (Global Shared scripts)
timestamps { // set the timestamps on the jenkins console
  timeout(40) { // Build timeout set to 40 mins
    node ('hyp-x') { // trigger jobs on x86_64 builds nodes
      // Applicable only on x86_64 for all the versions.
      // LF team has to install the newer version in Jenkins global config
      // Send an email to helpdesk@hyperledger.org to add newer version
      def nodeHome = tool 'nodejs-8.11.3'
      env.GO_VER = "1.10.4"
      env.GOPATH = "$WORKSPACE/gopath"
      env.GOROOT = "/opt/go/go${GO_VER}.linux.amd64"
      env.PATH = "$GOPATH/bin:$GOROOT/bin:/usr/local/bin:/usr/bin:/usr/local/sbin:/usr/sbin:~/npm/bin:${nodeHome}/bin:$PATH"
      def failure_stage = "none"
      // set MARCH value to amd64, s390x, ppc64le
      env.MARCH = sh(returnStdout: true, script: "uname -m | sed 's/x86_64/amd64/g'").trim()
      try {
        def ROOTDIR = pwd() // workspace dir (/w/workspace/<job_name>)
        stage('Clean Environment') {
          // delete working directory
          deleteDir()
          // Clean build environment before start the build
          fabBuildLibrary.cleanupEnv()
          // Display jenkins environment details
          fabBuildLibrary.envOutput()
        }
        stage('Checkout SCM') {
          // Get changes from gerrit
          fabBuildLibrary.cloneRefSpec('fabric-chaincode-evm')
          // Load properties from ci.properties file
          props = fabBuildLibrary.loadProperties()
        }
        // Run license-checks
        stage("Checks") {
          wrap([$class: 'AnsiColorBuildWrapper', 'colorMapName': 'xterm']) {
            try {
              dir("${ROOTDIR}/$PROJECT_DIR") {
                sh '''
                  echo "------> Run license, spelling, linter checks"
                  make basic-checks
                '''
              }
            }
            catch (err) {
              failure_stage = "basic-checks"
              currentBuild.result = 'FAILURE'
              throw err
            }
          }
        }
        // Run unit-tests (unit-tests)
        stage("Unit-Tests") {
          wrap([$class: 'AnsiColorBuildWrapper', 'colorMapName': 'xterm']) {
            try {
              dir("${ROOTDIR}/$PROJECT_DIR") {
                sh '''
                  echo "------> Run unit-tests"
                  make unit-tests
                '''
              }
            }
            catch (err) {
              failure_stage = "unit-tests"
              currentBuild.result = 'FAILURE'
              throw err
            }
          }
        }
        // Run integration tests (e2e tests)
        stage("Integration-Tests") {
          wrap([$class: 'AnsiColorBuildWrapper', 'colorMapName': 'xterm']) {
            try {
              dir("${ROOTDIR}/$PROJECT_DIR") {
                sh '''
                  echo "-------> Run integration-tests"
                  make integration-test
                '''
              }
            }
            catch (err) {
              failure_stage = "integration-test"
              currentBuild.result = 'FAILURE'
              throw err
            }
          }
        }
      } finally { // post build actions
      // Send notifications only for merge failures
      if (env.JOB_TYPE == "merge") {
        if (currentBuild.result == 'FAILURE') {
          // Send notification to rocketChat channel
          // Send merge build failure email notifications to the submitter
          sendNotifications(currentBuild.result, props["ROCKET_CHANNEL_NAME"])
        }
      }
      // Delete workspace when build is done
      cleanWs notFailBuild: true
    } // end finally block
    } // end node block
  } // end timeout block
} // end timestamps
