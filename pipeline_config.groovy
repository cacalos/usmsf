allow_scm_jenkinsfile = true
libraries{
  common
  make
  sonarqube{
    credential_id = "sonarqube"
    sonar_server = "SonarQubeJoy"
    enforce_quality_gate = false
  }
}
