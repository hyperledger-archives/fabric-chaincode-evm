// Copyright IBM Corp All Rights Reserved
//
// SPDX-License-Identifier: Apache-2.0
//
timeout(40) {
node ('hyp-x') { // trigger build on x86_64 node
   timestamps {
    try {
     def ROOTDIR = pwd() // workspace dir (/w/workspace/<job_name>)
     env.PROJECT_DIR = "gopath/src/github.com/hyperledger"
     env.GO_VER = "1.10.4"
     env.GOPATH = "$WORKSPACE/gopath"
     env.GOROOT = "/opt/go/go${GO_VER}.linux.amd64"
     def nodeHome = tool 'nodejs-8.11.3'
     def jobname = sh(returnStdout: true, script: 'echo ${JOB_NAME} | grep -q "verify" && echo patchset || echo merge').trim()
     env.PATH = "$GOPATH/bin:/usr/local/bin:/usr/bin:/usr/local/sbin:/usr/sbin:${nodeHome}/bin:$GOROOT/bin:$PATH"
     env.NODE_PATH = "/home/jenkins/npm/lib/node_modules"

     def failure_stage = "none"
// delete working directory
     deleteDir()
      stage("Fetch Patchset") {
          try {
             if (jobname == "patchset")  {
                   println "$GERRIT_REFSPEC"
                   println "$GERRIT_BRANCH"
                   // BASE_DIR points to gopath/src/github.com/hyperledger/fabric-chaincode-evm
                   // GIT_BASE points to git://cloud.hyperledger.org/mirror/fabric-chaincode-evm.git
                   checkout([
                       $class: 'GitSCM',
                       branches: [[name: '$GERRIT_REFSPEC']],
                       extensions: [[$class: 'RelativeTargetDirectory', relativeTargetDir: '$BASE_DIR'], [$class: 'CheckoutOption', timeout: 10]],
                       userRemoteConfigs: [[credentialsId: 'hyperledger-jobbuilder', name: 'origin', refspec: '$GERRIT_REFSPEC:$GERRIT_REFSPEC', url: '$GIT_BASE']]])
              } else {
                   // Clone fabric-chaincode-evm on merge
                   println "Clone $PROJECT repository"
                   checkout([
                       $class: 'GitSCM',
                       branches: [[name: 'refs/heads/$GERRIT_BRANCH']],
                       extensions: [[$class: 'RelativeTargetDirectory', relativeTargetDir: '$BASE_DIR']],
                       userRemoteConfigs: [[credentialsId: 'hyperledger-jobbuilder', name: 'origin', refspec: '+refs/heads/$GERRIT_BRANCH:refs/remotes/origin/$GERRIT_BRANCH', url: '$GIT_BASE']]])
              }
              dir("${ROOTDIR}/$PROJECT_DIR/$PROJECT"){
              sh '''
                 # Print last two commit details
                 echo
                 git log -n2 --pretty=oneline --abbrev-commit
                 echo
              '''
              }
          }
          catch (err) {
                 failure_stage = "Fetch patchset"
                 currentBuild.result = 'FAILURE'
                 throw err
          }
      }
// clean environment, get env data
      stage("CleanEnv - GetEnv") {
          try {
                 dir("${ROOTDIR}/$PROJECT_DIR/fabric-chaincode-evm/scripts/jenkins_scripts") {
                 sh './CI_Script.sh --clean_Environment --env_Info'
                 }
          }
          catch (err) {
                 failure_stage = "Clean Environment - Get Env Info"
                 currentBuild.result = 'FAILURE'
                 throw err
          }
      }


// Run license-checks
      stage("Checks") {
          try {
                 dir("${ROOTDIR}/$PROJECT_DIR/fabric-chaincode-evm") {
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

// Run unit-tests (unit-tests)
      stage("Unit-Tests") {
          try {
                 dir("${ROOTDIR}/$PROJECT_DIR/fabric-chaincode-evm") {
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
// Run integration tests (e2e tests)
      stage("Integration-Tests") {
          try {
                 dir("${ROOTDIR}/$PROJECT_DIR/fabric-chaincode-evm") {
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
           } finally {
              if (env.JOB_NAME == "fabric-chaincode-evm-merge-x86_64") {
                if (currentBuild.result == 'FAILURE') { // Other values: SUCCESS, UNSTABLE
                  rocketSend channel: 'fabric-evm', emoji: ':sob:', message: "Build Notification - STATUS: *${currentBuild.result}* - BRANCH: *${env.GERRIT_BRANCH}* - PROJECT: *${env.PROJECT}* - BUILD_URL - (<${env.BUILD_URL}|Open>)"
                }
              }
            } // finally
    } // timestamps
} // node
} // timeout
