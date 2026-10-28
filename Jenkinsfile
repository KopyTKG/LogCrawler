// Builds LogCrawler for the three desktop OSes and runs the tests.
//
// Agents are picked by label, so the Jenkins instance needs one agent each labelled `linux`,
// `windows` and `macos`, with Go (see go.mod), Node and yarn 1 on the PATH. A macOS binary can
// only be built on a Mac. Linux also needs gtk3 and webkit2gtk-4.1 development packages.
//
// Not signed or notarised: Windows shows a SmartScreen warning and macOS a Gatekeeper warning on
// first run. Signing needs the team's certificates, which belong in Jenkins credentials.
pipeline {
  agent none

  options {
    timestamps()
    disableConcurrentBuilds()
    buildDiscarder(logRotator(numToKeepStr: '20', artifactNumToKeepStr: '5'))
  }

  environment {
    CGO_ENABLED = '1'
    WAILS_VERSION = 'v3.0.0'
  }

  stages {
    stage('Test') {
      agent { label 'linux' }
      steps {
        sh 'go install -tags gtk3 github.com/wailsapp/wails/v3/cmd/wails3@${WAILS_VERSION}'
        sh 'yarn --cwd frontend install --frozen-lockfile'
        withEnv(["PATH+GOBIN=${env.HOME}/go/bin"]) {
          sh 'wails3 generate bindings -f "-tags gtk3" -ts -i'
        }
        sh 'go vet -tags gtk3 ./internal/... ./cmd/...'
        sh 'go test ./internal/...'
        sh 'yarn --cwd frontend run check'
        sh 'yarn --cwd frontend run lint'
        sh 'yarn --cwd frontend run format:check'
        sh 'yarn --cwd frontend run test'
        sh 'test -z "$(gofmt -l internal cmd *.go)"'
      }
    }

    stage('Build') {
      parallel {
        stage('Linux') {
          agent { label 'linux' }
          steps {
            sh 'go install -tags gtk3 github.com/wailsapp/wails/v3/cmd/wails3@${WAILS_VERSION}'
            sh 'yarn --cwd frontend install --frozen-lockfile'
            withEnv(["PATH+GOBIN=${env.HOME}/go/bin"]) {
              sh 'wails3 generate bindings -f "-tags gtk3" -ts -i'
            }
            sh 'yarn --cwd frontend build'
            sh 'go build -tags gtk3 -ldflags "-X main.version=${GIT_COMMIT}" -o bin/logcrawler-linux .'
            archiveArtifacts artifacts: 'bin/logcrawler-linux', fingerprint: true
          }
        }

        stage('Windows') {
          agent { label 'windows' }
          steps {
            bat 'go install github.com/wailsapp/wails/v3/cmd/wails3@%WAILS_VERSION%'
            bat 'yarn --cwd frontend install --frozen-lockfile'
            bat 'wails3 generate bindings -ts -i'
            bat 'yarn --cwd frontend build'
            bat 'go build -ldflags "-H windowsgui -X main.version=%GIT_COMMIT%" -o bin\\logcrawler.exe .'
            archiveArtifacts artifacts: 'bin/logcrawler.exe', fingerprint: true
          }
        }

        stage('macOS') {
          agent { label 'macos' }
          steps {
            sh 'go install github.com/wailsapp/wails/v3/cmd/wails3@${WAILS_VERSION}'
            sh 'yarn --cwd frontend install --frozen-lockfile'
            withEnv(["PATH+GOBIN=${env.HOME}/go/bin"]) {
              sh 'wails3 generate bindings -ts -i'
            }
            sh 'yarn --cwd frontend build'
            sh 'go build -ldflags "-X main.version=${GIT_COMMIT}" -o bin/logcrawler-macos .'
            archiveArtifacts artifacts: 'bin/logcrawler-macos', fingerprint: true
          }
        }
      }
    }
  }
}
