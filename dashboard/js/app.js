document.addEventListener("DOMContentLoaded", () => {
  const totalBetsElem = document.getElementById("total-bets");
  const betsPerSecondElem = document.getElementById("bets-per-second");
  const totalValueElem = document.getElementById("total-value");
  const dbFailuresPerSecondElem = document.getElementById("db-failures-per-second");
  const redisFailuresPerSecondElem = document.getElementById("redis-failures-per-second");
  const dbFailuresElem = document.getElementById("db-failures");
  const redisFailuresElem = document.getElementById("redis-failures");
  const connectionStatus = document.getElementsByClassName("status-pill")[0];
  const rowsCountElem = document.getElementById("bets-table-row-count");

  const socket = new WebSocket("ws://localhost:8081/ws");

  let intervalId = -1;

  function showTableRowsCount() {
    fetch("http://localhost:8080/stats")
      .then((res) => res.json())
      .then((data) => {
        rowsCountElem.textContent = `Total Bets in the table: ${data.total_bets}`;

        rowsCountElem.className =
          "status-pill " + (data.total_bets === 0 ? "error" : data.total_bets < 10 ? "warning" : "success");
      })
      .catch((error) => {
        console.log("Fetch error:", error);
      });
  }

  function stopShowTableRowsCount() {
    if (intervalId != -1) {
      clearInterval(intervalId);
      rowsCountElem.textContent = "CONNECTION DISABLED";
    }
  }

  socket.onopen = () => {
    connectionStatus.textContent = "Connected";
    connectionStatus.classList.remove("waiting");
    connectionStatus.classList.add("success");
    console.log("WebSocket connection established.");
    showTableRowsCount();
    intervalId = setInterval(showTableRowsCount, 30000);
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
    stopShowTableRowsCount();
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
    stopShowTableRowsCount();
  };
});
