#!/bin/bash
set +e  # Continue on errors

COLOR_CYAN="\033[0;36m"
COLOR_RESET="\033[0m"

RUN_CMD="go run main.go"
DEBUG_CMD="dlv debug ./main.go --listen=0.0.0.0:2347 --api-version=2 --output /tmp/__debug_bin --headless"

echo -e "${COLOR_CYAN}
   ____              ____
  |  _ \  _____   __/ ___| _ __   __ _  ___ ___
  | | | |/ _ \ \ / /\___ \| '_ \ / _\` |/ __/ _ \\
  | |_| |  __/\ V /  ___) | |_) | (_| | (_|  __/
  |____/ \___| \_/  |____/| .__/ \__,_|\___\___|
                          |_|
${COLOR_RESET}
Welcome to your development container!
This is how you can work with it:
- Run \`${COLOR_CYAN}${RUN_CMD}${COLOR_RESET}\` to start the k8sgpt-controller
- ${COLOR_CYAN}Files will be synchronized${COLOR_RESET} between your local machine and this container

If you wish to run k8sgpt-controller in debug mode with delve, run:
  \`${COLOR_CYAN}${DEBUG_CMD}${COLOR_RESET}\`
  Wait until the \`${COLOR_CYAN}API server listening at: [::]:2347${COLOR_RESET}\` message appears
  Start the \"Debug (localhost:2347)\" configuration in VSCode to connect your debugger session.
  ${COLOR_CYAN}Note:${COLOR_RESET} k8sgpt-controller won't start until you connect with the debugger.
  ${COLOR_CYAN}Note:${COLOR_RESET} k8sgpt-controller will be stopped once you detach your debugger session.

${COLOR_CYAN}TIP:${COLOR_RESET} hit an up arrow on your keyboard to find the commands mentioned above :) 
"
# add useful commands to the history for convenience
export HISTFILE=/tmp/.bash_history
history -s $DEBUG_CMD
history -s $RUN_CMD
history -a

# hide "I have no name!" from the bash prompt
bash --init-file <(echo "export PS1=\"\\H:\\W\\$ \"")