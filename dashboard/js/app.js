document.addEventListener("DOMContentLoaded", () => {
  const totalBetsElem = document.getElementById("total-bets");
  const betsPerSecondElem = document.getElementById("bets-per-second");
  const totalValueElem = document.getElementById("total-value");
  const dbFailuresPerSecondElem = document.getElementById("db-failures-per-second");
  const redisFailuresPerSecondElem = document.getElementById("redis-failures-per-second");
  const dbFailuresElem = document.getElementById("db-failures");
  const redisFailuresElem = document.getElementById("redis-failures");
  const connectionStatus = document.getElementsByClassName("status-pill")[0];

  const socket = new WebSocket("ws://localhost:8081/ws");

  socket.onopen = () => {
    connectionStatus.textContent = "Connected";
    connectionStatus.classList.remove("waiting");
    connectionStatus.classList.add("success");
    console.log("WebSocket connection established.");
  };

  socket.onmessage = (event) => {
    const stats = JSON.parse(event.data);
    totalBetsElem.textContent = stats.total_bets.toLocaleString();
    betsPerSecondElem.textContent = stats.bets_per_second.toFixed(0);
    totalValueElem.textContent = `$${stats.total_value.toLocaleString(undefined, {
      minimumFractionDigits: 2,
      maximumFractionDigits: 2,
    })}`;
    dbFailuresPerSecondElem.textContent = stats.db_failures_per_second.toLocaleString();
    redisFailuresPerSecondElem.textContent = stats.redis_failures_per_second.toLocaleString();
    dbFailuresElem.textContent = stats.db_failures.toLocaleString();
    redisFailuresElem.textContent = stats.redis_failures.toLocaleString();
  };

  socket.onclose = () => {
    connectionStatus.textContent = "Connection closed";
    connectionStatus.classList.remove("waiting");
    connectionStatus.classList.add("error");
    console.log("WebSocket connection closed");
  };

  socket.onerror = (error) => {
    connectionStatus.textContent = "Error";
    connectionStatus.classList.remove("waiting");
    connectionStatus.classList.add("error");

    console.error(
      "WebSocket error:",
      error,
      "\n\nCheck if port forwarding is set up (kubectl port-forward svc/nats-service 8081:8081)"
    );
  };
});
