allow_scm_jenkinsfile = true
libraries{
    common{
        makefile_config = '''
PKG_LIST := $(shell go list ./... | grep -vE &quot;test|cmd\\/httpif|cmd\\/nrfclient|cmd\\/usmsfperf&quot;)
GO_FILES := $(shell find . -name &quot;*.go&quot; | grep -v /vendor/ | grep -v _test.go)
MAKEIMAGE_SHELL := sh ./cmd/usmsf/jenkins_build.sh
'''
    }
	helm{
        overriding = "resources/helm_charts/dish_samsung/overriding-values/uangel/t2x-master1/ci-cd-test/values.yaml"
        name_space = "testjks"
        name = "usmsf-jks"
        service = "resources/helm_charts/dish_samsung/dish-smsf-1.2.1.tgz"
	}
    make
    gitfile
    email{
        final_stage_sendmail = false
        fail_mail_lists = "cacalos1@uangel.com"
        success_mail_lists = "cacalos@korea.com"
    }
    sonarqube{
        credential_id = "sonarqube"
        sonar_server = "SonarQubeCaca"
        enforce_quality_gate = false
    }
}
