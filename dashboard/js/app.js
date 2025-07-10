document.addEventListener("DOMContentLoaded", () => {
  const totalBetsElem = document.getElementById("total-bets");
  const betsPerSecondElem = document.getElementById("bets-per-second");
  const totalValueElem = document.getElementById("total-value");

  const socket = new WebSocket("ws://localhost:8080/ws");

  socket.onopen = () => {
    console.log("WebSocket connection established");
  };

  socket.onmessage = (event) => {
    const stats = JSON.parse(event.data);
    totalBetsElem.textContent = stats.total_bets.toLocaleString();
    betsPerSecondElem.textContent = stats.bets_per_second.toFixed(0);
    totalValueElem.textContent = `$${stats.total_value.toLocaleString(undefined, {
      minimumFractionDigits: 2,
      maximumFractionDigits: 2,
    })}`;
  };

  socket.onclose = () => {
    console.log("WebSocket connection closed");
  };

  socket.onerror = (error) => {
    console.error("WebSocket error:", error);
  };
});
