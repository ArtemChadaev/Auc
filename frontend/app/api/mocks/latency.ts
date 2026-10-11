/**
 * Имитация задержки сети и случайных сбоев.
 */

export const SIMULATION_CONFIG = {
  minLatencyMs: 120,
  maxLatencyMs: 350,
  simulateNetworkFailures: false,
};

export async function simulateLatency(customMs?: number): Promise<void> {
  const delay =
    customMs ??
    (Math.floor(
      Math.random() * (SIMULATION_CONFIG.maxLatencyMs - SIMULATION_CONFIG.minLatencyMs)
    ) + SIMULATION_CONFIG.minLatencyMs);

  await new Promise((resolve) => setTimeout(resolve, delay));

  if (SIMULATION_CONFIG.simulateNetworkFailures && Math.random() < 0.05) {
    throw new Error('Сетевой сбой (эмуляция)');
  }
}
