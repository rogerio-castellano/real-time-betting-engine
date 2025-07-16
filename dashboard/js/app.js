document.addEventListener("DOMContentLoaded", () => {
  const totalBetsElem = document.getElementById("total-bets");
  const betsPerSecondElem = document.getElementById("bets-per-second");
  const totalValueElem = document.getElementById("total-value");
  const dbFailuresPerSecondElem = document.getElementById("db-failures-per-second");
  const redisFailuresPerSecondElem = document.getElementById("redis-failures-per-second");
  const dbFailuresElem = document.getElementById("db-failures");
  const redisFailuresElem = document.getElementById("redis-failures");
  const totalOddsElem = document.getElementById("total-odds");
  const pendingBetsElem = document.getElementById("pending-bets");
  const connectionStatus = document.getElementsByClassName("status-pill")[0];
  const rowsCountElem = document.getElementById("bets-table-row-count");
  const refreshIntervalElem = document.getElementById("refresh-interval");
  const statusBannerElem = document.getElementById("statusBanner");
  const reconnectBtnElem = document.getElementById("reconnectBtn");

  const refreshIntervalInSeconds = 15;
  const reconnectIntervalInSeconds = 15;
  const maxRetries = 5;
  let retryCount = 0;
  let intervalId = -1;
  let ingestedBetsCount = 0;

  refreshIntervalElem.textContent = refreshIntervalInSeconds;

  connectSocket();

  function connectSocket() {
    const socket = new WebSocket("ws://localhost:8081/ws");

    if (retryCount >= maxRetries) {
      console.error(`❌ Max retries (${maxRetries}) reached. Giving up.`);
      showReconnectButton();
      return;
    }

    socket.onopen = () => {
      connectionStatus.textContent = "Connected";
      connectionStatus.classList.remove("waiting");
      connectionStatus.classList.add("success");
      console.log("✅ WebSocket connection established.");
      retryCount = 0;
      hideReconnectButton();
      showTableRowsCount();
      intervalId = setInterval(showTableRowsCount, refreshIntervalInSeconds * 1000);
    };

    socket.onmessage = (event) => {
      const stats = JSON.parse(event.data);
      ingestedBetsCount = stats.total_bets;
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

      retryCount++;
      console.warn(`⚠️ Connection lost. Retry ${retryCount} of ${maxRetries} in ${reconnectIntervalInSeconds}s...`);
      connectionStatus.classList.value = "status-pill";
      connectionStatus.classList.add("waiting");
      connectionStatus.textContent = "Retrying...";
      setTimeout(connectSocket, reconnectIntervalInSeconds * 1000);
    };

    socket.onerror = (error) => {
      connectionStatus.textContent = "Error";
      connectionStatus.classList.remove("waiting");
      connectionStatus.classList.add("error");

      console.error("WebSocket error:", error, "\n\nCheck if the service stats-aggregator is available");
      stopShowTableRowsCount();
      socket.close();
    };

    reconnectBtnElem.addEventListener("click", () => {
      retryCount = 0;
      hideReconnectButton();
      connectSocket();
    });

    function showReconnectButton() {
      reconnectBtnElem.style.display = reconnectBtnElem.style.display = "block";
      statusBannerElem.style.display = statusBannerElem.style.display = "block";
      connectionStatus.style.display = connectionStatus.style.display = "none";
    }

    function hideReconnectButton() {
      reconnectBtnElem.style.display = reconnectBtnElem.style.display = "none";
      statusBannerElem.style.display = statusBannerElem.style.display = "none";
      connectionStatus.style.display = connectionStatus.style.display = "block";
    }
  }

  function showTableRowsCount() {
    fetch("http://localhost:8082/stats")
      .then((res) => res.json())
      .then((data) => {
        betsTableRowCount = data.bets_table_row_count;
        rowsCountElem.textContent = `Total Bets in the table: ${betsTableRowCount}`;

        const errorsRate = (ingestedBetsCount - betsTableRowCount) / ingestedBetsCount;

        const thresholds = [
          { limit: 0.01, color: "color7" },
          { limit: 0.008, color: "color6" },
          { limit: 0.006, color: "color5" },
          { limit: 0.004, color: "color4" },
          { limit: 0.002, color: "color3" },
          { limit: 0, color: "color2" },
        ];

        let color = "color1"; // default for exact match

        for (const { limit, color: c } of thresholds) {
          if (errorsRate > limit) {
            color = c;
            break;
          }
        }

        rowsCountElem.className = `status-pill ${color}`;
        totalOddsElem.textContent = data.total_odds.toLocaleString();
        pendingBetsElem.textContent = data.pending_bets.toLocaleString();
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
});

function formattedTime(date) {
  const hours = String(date.getHours()).padStart(2, "0");
  const minutes = String(date.getMinutes()).padStart(2, "0");
  const seconds = String(date.getSeconds()).padStart(2, "0");

  return `${hours}:${minutes}:${seconds}`;
}
