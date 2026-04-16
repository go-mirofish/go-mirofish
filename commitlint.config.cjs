/** @type {import('@commitlint/types').UserConfig} */
module.exports = {
  extends: ['@commitlint/config-conventional'],
  rules: {
    'type-enum': [
      2,
      'always',
      [
        'feat',
        'fix',
        'docs',
        'refactor',
        'test',
        'chore',
        'ci',
        'perf',
        'build',
        'revert',
      ],
    ],
    'scope-enum': [
      2,
      'always',
      [
        'gateway',
        'python',
        'frontend',
        'docs',
        'ci',
        'config',
        'deps',
        'release',
        'readme',
      ],
    ],
    'scope-empty': [2, 'never'],
    'subject-max-length': [2, 'always', 72],
  },
};
