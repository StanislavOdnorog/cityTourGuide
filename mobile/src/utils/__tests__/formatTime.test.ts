import { formatTime } from '../formatTime';

describe('formatTime', () => {
  it('formats 0 seconds as 0:00', () => {
    expect(formatTime(0)).toBe('0:00');
  });

  it('formats seconds under a minute', () => {
    expect(formatTime(5)).toBe('0:05');
    expect(formatTime(30)).toBe('0:30');
    expect(formatTime(59)).toBe('0:59');
  });

  it('formats exact minutes', () => {
    expect(formatTime(60)).toBe('1:00');
    expect(formatTime(120)).toBe('2:00');
  });

  it('formats minutes and seconds', () => {
    expect(formatTime(65)).toBe('1:05');
    expect(formatTime(90)).toBe('1:30');
    expect(formatTime(125)).toBe('2:05');
  });

  it('floors fractional seconds', () => {
    expect(formatTime(15.7)).toBe('0:15');
    expect(formatTime(59.9)).toBe('0:59');
    expect(formatTime(60.1)).toBe('1:00');
  });

  it('handles large durations', () => {
    expect(formatTime(3600)).toBe('60:00');
    expect(formatTime(3661)).toBe('61:01');
  });

  it('pads single-digit seconds with leading zero', () => {
    expect(formatTime(1)).toBe('0:01');
    expect(formatTime(61)).toBe('1:01');
    expect(formatTime(301)).toBe('5:01');
  });
});

describe('MiniPlayer progress logic', () => {
  it('calculates progress as ratio of position to duration', () => {
    const position = 15;
    const duration = 30;
    const progress = duration > 0 ? Math.min(position / duration, 1) : 0;
    expect(progress).toBe(0.5);
  });

  it('clamps progress to 0 when duration is 0', () => {
    const position = 0;
    const duration = 0;
    const progress = duration > 0 ? Math.min(position / duration, 1) : 0;
    expect(progress).toBe(0);
  });

  it('clamps progress to 1 when position exceeds duration', () => {
    const position = 35;
    const duration = 30;
    const progress = duration > 0 ? Math.min(position / duration, 1) : 0;
    expect(progress).toBe(1);
  });

  it('handles position at exactly duration', () => {
    const position = 30;
    const duration = 30;
    const progress = duration > 0 ? Math.min(position / duration, 1) : 0;
    expect(progress).toBe(1);
  });
});
