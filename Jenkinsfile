pipeline {
    agent any

    environment {
        PROJECT_DIR = '/var/www/vhosts/entegra.goadisyon.com/httpdocs'
        GO_BIN      = '/usr/local/go/bin/go'
        SERVICE     = 'go-entegra'
    }

    triggers {
        githubPush()
    }

    stages {

        stage('Checkout') {
            steps {
                echo "Branch: ${env.GIT_BRANCH}"
                checkout scm
            }
        }

        stage('Build') {
            steps {
                sh '$GO_BIN build -o ./entegra ./cmd/main.go'
                echo "Build OK"
            }
        }

        stage('Deploy') {
            steps {
                sh '''
                    cp -f ./entegra $PROJECT_DIR/entegra
                    cp -rf ./web    $PROJECT_DIR/web
                    systemctl restart $SERVICE
                    sleep 3
                    if systemctl is-active --quiet $SERVICE; then
                        echo "Service running OK"
                    else
                        echo "Service FAILED"
                        journalctl -u $SERVICE -n 50 --no-pager
                        exit 1
                    fi
                '''
            }
        }

    }

    post {
        success {
            echo "Deploy OK — ${env.GIT_BRANCH} @ ${env.GIT_COMMIT?.take(7)}"
        }
        failure {
            echo "Deploy FAILED — ${env.GIT_BRANCH} @ ${env.GIT_COMMIT?.take(7)}"
        }
    }
}
