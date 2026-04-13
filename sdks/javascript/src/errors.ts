export class BaconError extends Error {
  constructor(
    public statusCode: number,
    public type: string,
    public title: string,
    public detail: string,
    public instance: string,
  ) {
    super(`${title} (${statusCode}): ${detail}`);
    this.name = 'BaconError';
  }
}
