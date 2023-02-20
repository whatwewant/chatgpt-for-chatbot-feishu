#!/bin/zmicro

help() {
  echo "Usage:"
  echo "  chatgpt-for-chatbot-feishu"
}

core() {
  if [ "$1" = "-h" ] || [ "$1" = "--help" ]; then
    help
    exit 0
  fi

  dotenv::try_load

  local PORT=${PORT:-8080}
  local API_PATH=${API_PATH:-"/"}

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
      zmicro ngrok http --subdomain "$ngrok_subdomain" ${PORT} --log $ngrok_log >>$ngrok_log 2>&1 &
    else
      zmicro ngrok http ${PORT} --log $ngrok_log >>$ngrok_log 2>&1 &
    fi

    log::info "[$(timestamp)] starting ngrok ..."
    # sleep 3

    local ngrok_url=""
    while [ -z "$ngrok_url" ]; do
      sleep 1
      
      log::info "[$(timestamp)] checking whether ngrok connected ..."
      ngrok_url=$(cat $ngrok_log | grep "url=" | awk -F '=' '{print $8}')
      if [ -n "$ngrok_url" ]; then
        break
      fi

      if [ "$DEBUG" = "true" ]; then
        log::info "[$(timestamp)] show ngrok connection info start ..."
        cat $ngrok_log
        log::info "[$(timestamp)] show ngrok connection info end ..."
      fi
    done

    log::info "[$(timestamp)] ngrok url: $(color::green $ngrok_url)"

    export SITE_URL=$ngrok_url
  fi

  log::info "[$(timestamp)] starting chatgpt for chatbot feishu ..."
  chatgpt-for-chatbot-feishu
}

run() {
  core "$@"
}

run "$@"
