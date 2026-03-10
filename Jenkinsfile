pipeline {
    agent any

    parameters {
        choice(
            name: 'MODULE',
            choices: [
                'transaction-core', 'transaction-data', 'caching', 'configuration', 'db',
                'iam', 'iso-messaging', 'job-engine', 'lookup', 'member-data-core',
                'messaging', 'object-storage', 'operator-data-core', 'qr', 'security', 'shared'
            ],
            description: 'Module to build and publish to Artifactory'
        )
        string(
            name: 'BRANCH',
            defaultValue: 'main',
            description: 'Git branch / tag / commit to build'
        )
        string(
            name: 'VERSION_OVERRIDE',
            defaultValue: '',
            description: 'Force specific version (leave empty to read from build.gradle)'
        )
        booleanParam(
            name: 'SKIP_TESTS',
            defaultValue: true,
            description: 'Skip unit & integration tests (-x test)'
        )
    }

    environment {
        ARTIFACTORY_URL      = 'https://artifacts.finpos.global/artifactory'
        GITHUB_CRED          = 'github-hardened-token'
        ARTIFACTORY_CREDENTIAL = 'ARTIFACTORY_CRED'
    }

    stages {
        stage('Initialize') {
            steps {
                script {
                    def repoMap = [
                        'transaction-core' : 'https://github.com/payment-engine.git',
                        'transaction-data' : 'https://github.com/payment-engine.git',
                        'job-engine'       : 'https://github.com/job-engine.git'
                    ]

                    def pathMap = [
                        'transaction-core'     : 'libs-release-local/global/transaction-core',
                        'transaction-data'     : 'libs-release-local/global/transaction-data',
                        'caching'              : 'libs-release-local/global/caching',
                        'configuration'        : 'libs-release-local/global/configuration',
                        'db'                   : 'libs-release-local/global/db',
                        'iam'                  : 'libs-release-local/global/iam',
                        'iso-messaging'        : 'libs-release-local/global/iso/iso-messaging',
                        'job-engine'           : 'libs-release-local/global/job-engine/core',
                        'lookup'               : 'libs-release-local/global/lookup',
                        'member-data-core'     : 'libs-release-local/global/member-data-core',
                        'messaging'            : 'libs-release-local/global/messaging',
                        'object-storage'       : 'libs-release-local/global/object-storage',
                        'operator-data-core'   : 'libs-release-local/global/operator-data-core',
                        'qr'                   : 'libs-release-local/global/qr',
                        'security'             : 'libs-release-local/global/security',
                        'shared'               : 'libs-release-local/global/shared'
                    ]

                    env.REPO_URL = repoMap[params.MODULE] ?: 'https://github.com/demo-repo/infrastructure.git'
                    env.ARTIFACTORY_BASE = pathMap[params.MODULE]

                    if (!env.ARTIFACTORY_BASE) {
                        error "No Artifactory path mapping found for module: ${params.MODULE}"
                    }

                    env.GRADLE_PROJECT = params.MODULE == 'job-engine' ? ':core' : ":${params.MODULE}"

                    echo """
                    ╔═══════════════════════════════════════════════════════════════╗
                    ║                  PUBLISH CONFIGURATION                        ║
                    ╠═══════════════════════════════════════════════════════════════╣
                    ║ Module           : ${params.MODULE.padRight(35)} ║
                    ║ Repository       : ${env.REPO_URL} ║
                    ║ Gradle project   : ${env.GRADLE_PROJECT} ║
                    ║ Target Artifactory : ${env.ARTIFACTORY_BASE} ║
                    ║ Branch           : ${params.BRANCH} ║
                    ╚═══════════════════════════════════════════════════════════════╝
                    """
                }
            }
        }

        stage('Checkout') {
            steps {
                cleanWs()
                git branch: params.BRANCH,
                    credentialsId: env.GITHUB_CRED,
                    url: env.REPO_URL
            }
        }

        stage('Determine Version') {
            steps {
                script {
                    def buildGradle = "${params.MODULE == 'job-engine' ? 'job-engine/core' : params.MODULE}/build.gradle"
                    if (!fileExists(buildGradle)) {
                        error "build.gradle not found → ${buildGradle}"
                    }

                    def content = readFile(buildGradle)
                    def versionMatcher = (content =~ /(?m)^\s*version\s*[=]\s*["']?([^"'\s;]+)["']?\s*(?:\/\/.*)?$/)

                    def detected = versionMatcher.find() ? versionMatcher[0][1].trim() : null

                    if (params.VERSION_OVERRIDE?.trim()) {
                        env.MODULE_VERSION = params.VERSION_OVERRIDE.trim()
                        echo "Using overridden version: ${env.MODULE_VERSION}"
                    } else if (detected) {
                        env.MODULE_VERSION = detected
                        echo "Detected version from build.gradle: ${env.MODULE_VERSION}"
                    } else {
                        error "Impossible to determine version! No version in build.gradle & no override."
                    }
                }
            }
        }

        stage('Prevent Overwrite - Check Artifactory') {
            steps {
                script {
                    def versionPath = "${env.ARTIFACTORY_BASE}/${env.MODULE_VERSION}"

                    withCredentials([usernamePassword(
                        credentialsId: env.ARTIFACTORY_CREDENTIAL,
                        usernameVariable: 'ART_USER',
                        passwordVariable: 'ART_PASS'
                    )]) {
                        def status = sh(script: """
                            curl -s -o /dev/null -w "%{http_code}" \
                                -u "\${ART_USER}:\${ART_PASS}" \
                                "${env.ARTIFACTORY_URL}/api/storage/${versionPath}"
                        """, returnStdout: true).trim()

                        if (status == '200') {
                            error "Version already exists in Artifactory: ${versionPath}"
                        }
                        echo "→ Version folder does NOT exist → good to go"
                    }
                }
            }
        }

        stage('Prepare Gradle Configuration') {
            steps {
                withCredentials([usernamePassword(
                    credentialsId: env.ARTIFACTORY_CREDENTIAL,
                    usernameVariable: 'ART_USER',
                    passwordVariable: 'ART_PASS'
                )]) {
                    script {
                        def workingSubdir = params.MODULE == 'job-engine' ? 'job-engine/core' : params.MODULE
                        def propsFile = "${workingSubdir}/gradle.properties"
                        def props = fileExists(propsFile) ? readFile(propsFile) : ""

                        props = props.replaceAll(/(?s)#.*Auto-injected.*?(?=\n|$)/, '').trim()
                        if (props) props += "\n\n"
                        props += "# Auto-injected by Jenkins - ${new Date().format('yyyy-MM-dd HH:mm:ss')}\n" +
                                 "artifactoryUrl=${env.ARTIFACTORY_URL}\n" +
                                 "artifactoryUsername=${ART_USER}\n" +
                                 "artifactoryPassword=${ART_PASS}\n"
                        writeFile file: propsFile, text: props
                    }
                }
            }
        }

        stage('Build & Publish') {
            steps {
                script {
                    boolean isWin = isUnix() == false
                    def gradlew = isWin ? 'gradlew.bat' : './gradlew'
                    def testFlag = params.SKIP_TESTS ? '-x test' : ''
                    def refreshDeps = '--refresh-dependencies'

                    if (isWin) {
                        if (!fileExists('gradlew.bat')) error "gradlew.bat missing"
                        bat """
                            ${gradlew} clean ${env.GRADLE_PROJECT}:build ${env.GRADLE_PROJECT}:publish \
                                ${testFlag} ${refreshDeps} --no-daemon --stacktrace --console plain
                        """
                    } else {
                        if (!fileExists('gradlew')) error "gradlew missing"
                        sh """
                            chmod +x gradlew 2>/dev/null || true
                            ${gradlew} clean ${env.GRADLE_PROJECT}:build ${env.GRADLE_PROJECT}:publish \
                                ${testFlag} ${refreshDeps} --no-daemon --stacktrace --console plain
                        """
                    }
                }
            }
        }
    }

    post {
        always {
            cleanWs()
        }
        success {
            echo """
            ╔═══════════════════════════════════════════════════════════════╗
            ║                      PUBLISH SUCCESS                          ║
            ╠═══════════════════════════════════════════════════════════════╣
            ║ Module           : ${params.MODULE}                            ║
            ║ Version          : ${env.MODULE_VERSION}                       ║
            ║ Published to     : ${env.ARTIFACTORY_BASE}/${env.MODULE_VERSION} ║
            ╚═══════════════════════════════════════════════════════════════╝
            """
        }
        failure {
            echo """
            ╔═══════════════════════════════════════════════════════════════╗
            ║                   BUILD / PUBLISH FAILED                      ║
            ╠═══════════════════════════════════════════════════════════════╣
            ║ Module           : ${params.MODULE}                            ║
            ║ Version          : ${env.MODULE_VERSION ?: 'unknown'}          ║
            ╚═══════════════════════════════════════════════════════════════╝
            """
        }
    }
}
