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

  if [ "$TUNNEL_ENABLE" = "true" ]; then
    local tunnel_type="$TUNNEL_TYPE"
    local tunnel_auth_token="$TUNNEL_AUTH_TOKEN"
    local tunnel_subdomain="$TUNNEL_SUBDOMAIN"
    local tunnel_log=$(os::tmp_file)

    log::info "[$(timestamp)] enable tunnel $tunnel_type (logfile: $tunnel_log)..."

    # TUNNEL NGROK
    if [ "$tunnel_type" = "ngrok" ]; then
      if [ -n "$tunnel_subdomain" ] && [ -z "$tunnel_auth_token" ]; then
        log::error "[$(timestamp)] tunnel_auth_token is required when use tunnel_subdomain"
        return 1
      fi

      if [ -n "$tunnel_auth_token" ]; then
        zmicro ngrok config add-authtoken $tunnel_auth_token
      fi

      if [ -n "$tunnel_subdomain" ]; then
        zmicro ngrok http --subdomain "$tunnel_subdomain" ${PORT} --log $tunnel_log >>$tunnel_log 2>&1 &
      else
        zmicro ngrok http ${PORT} --log $tunnel_log >>$tunnel_log 2>&1 &
      fi

      log::info "[$(timestamp)] starting ngrok ..."
      # sleep 3

      local url=""
      while [ -z "$url" ]; do
        sleep 1

        log::info "[$(timestamp)] checking whether ngrok connected ..."
        url=$(cat $tunnel_log | grep "url=" | head -n 1 | awk -F '=' '{print $8}')
        if [ -n "$url" ]; then
          break
        fi

        if [ "$DEBUG" = "true" ]; then
          log::info "[$(timestamp)] show ngrok connection info start ..."
          cat $tunnel_log
          log::info "[$(timestamp)] show ngrok connection info end ..."
        fi
      done

      log::info "[$(timestamp)] ngrok url: $(color::green $url)"

      export SITE_URL=$url
    else if [ "$TUNNEL_TYPE" = "cpolar" ]; then
      if [ -n "$tunnel_subdomain" ] && [ -z "$tunnel_auth_token" ]; then
        log::error "[$(timestamp)] tunnel_auth_token is required when use tunnel_subdomain"
        return 1
      fi

      if [ -n "$tunnel_auth_token" ]; then
        zmicro cpolar config authtoken $tunnel_auth_token
      fi

      if [ -n "$tunnel_subdomain" ]; then
        zmicro cpolar http --subdomain "$tunnel_subdomain" ${PORT} --log $tunnel_log >>$tunnel_log 2>&1 &
      else
        zmicro cpolar http ${PORT} --log $tunnel_log >>$tunnel_log 2>&1 &
      fi

      log::info "[$(timestamp)] starting cpolar ..."
      # sleep 3

      local cpolar_url=""
      while [ -z "$cpolar_url" ]; do
        sleep 1

        log::info "[$(timestamp)] checking whether cpolar connected ..."
        cpolar_url=$(cat $tunnel_log | grep "established" | grep "https" | awk -F 'at ' '{print $2}')
        if [ -n "$cpolar_url" ]; then
          break
        fi

        if [ "$DEBUG" = "true" ]; then
          log::info "[$(timestamp)] show cpolar connection info start ..."
          cat $tunnel_log
          log::info "[$(timestamp)] show cpolar connection info end ..."
        fi
      done

      log::info "[$(timestamp)] cpolar url: $(color::green $cpolar_url)"

      export SITE_URL=$cpolar_url
    fi
  fi

  log::info "[$(timestamp)] starting chatgpt for chatbot feishu ..."
  chatgpt-for-chatbot-feishu
}

run() {
  core "$@"
}

run "$@"
