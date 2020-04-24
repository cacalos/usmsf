libraries{
    common{
        makefile_name = "goMakefile"
        makefile_config = '''
PKG_LIST := $(shell go list ./... | grep -vE &quot;test|cmd\\/httpif|cmd\\/nrfclient|cmd\\/usmsfperf&quot;)
GO_FILES := $(shell find . -name &quot;*.go&quot; | grep -v /vendor/ | grep -v _test.go)
MAKEIMAGE_SHELL := sh ./cmd/usmsf/jenkins_build.sh
'''
    }
    make
    docker {
		remove_local_image = false
		registry = "docker.io/joyddung"
		cred = "docker-hub"
		repo_path_prefix = ""
		build_strategy = "dockerfile"
	}
    email{
        final_stage_sendmail = false
        fail_mail_lists = "cacalos1@uangel.com"
        success_mail_lists = "cacalos@korea.com"
    }
    kubernetes{
        helm_configuration_repository ="https://github.com/cacalos/dish-smsf.git"
        helm_configuration_repository_credential = "github-cacalos-pwd"
    }
}
