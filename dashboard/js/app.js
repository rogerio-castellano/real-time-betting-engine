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
  const refreshIntervalElem = document.getElementById("refresh-interval");

  const socket = new WebSocket("ws://localhost:8081/ws");
  const refreshIntervalInSeconds = 15;
  let intervalId = -1;
  let processedBetsCount = 0;

  refreshIntervalElem.textContent = refreshIntervalInSeconds;

  function showTableRowsCount() {
    fetch("http://localhost:8082/stats")
      .then((res) => res.json())
      .then((data) => {
        rowsCountElem.textContent = `Total Bets in the table: ${data.total_bets}`;

        rowsCountElem.className = "status-pill " + (data.total_bets !== processedBetsCount ? "error" : "success");
      })
      .catch((error) => {
        rowsCountElem.textContent = "Error (" + error + ") at " + formattedTime(new Date());
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
    intervalId = setInterval(showTableRowsCount, refreshIntervalInSeconds * 1000);
  };

  socket.onmessage = (event) => {
    const stats = JSON.parse(event.data);
    processedBetsCount = stats.total_bets;
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

    console.error("WebSocket error:", error, "\n\nCheck if the service stats-aggregator is available");
    stopShowTableRowsCount();
  };
});

function formattedTime(date) {
  const hours = String(date.getHours()).padStart(2, "0");
  const minutes = String(date.getMinutes()).padStart(2, "0");
  const seconds = String(date.getSeconds()).padStart(2, "0");

  return `${hours}:${minutes}:${seconds}`;
}
