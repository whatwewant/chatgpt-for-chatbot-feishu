#!/bin/zmicro

help() {
  echo "Usage:"
  echo "  chatgpt-for-chatbot-feishu"
}

core() {
  if [ "$1" = "-h" ] || [ "$1" = "--help" ] || [ -z "$1" ]; then
    help
    exit 0
  fi

  dotenv::try_load

  local PORT=${PORT:-8080}

  log::info "[$(timestamp)] run chatgpt for chatbot feishu with zmicro ..."

  if [ "$NGROK_ENABLE" = "true" ]; then
    local ngrok_log=$(os::tmp_file)
    local ngrok_auth_token=${NGROK_AUTH_TOKEN}
    local ngrok_subdomain=${NGROK_SUBDOMAIN}

    log::info "[$(timestamp)] enable ngrok (logfile: $ngrok_log)..."

    if [ -n "$ngrok_subdomain" ] && [ -z "$ngrok_auth_token" ]; then
      log::error "[$(timestamp)] NGROK_AUTH_TOKEN is required when use NGROK_SUBDOMAIN"
      return 1
    fi

    if [ -n "$ngrok_auth_token" ]; then
      zmicro ngrok config add-authtoken $ngrok_auth_token
    fi

    if [ -n "$ngrok_subdomain" ]; then
      zmicro ngrok http --subdomain "$ngrok_subdomain" ${PORT} >>$ngrok_log
    else
      zmicro ngrok http ${PORT} >>$ngrok_log
    fi

    log::info "[$(timestamp)] starting ngrok ..."
    sleep 3
    cat $ngrok_log
  fi

  log::info "[$(timestamp)] starting chatgpt for chatbot feishu ..."
  chatgpt-for-chatbot-feishu
}

run() {
  core "$@"
}

run "$@"
