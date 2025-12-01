import type { CodegenConfig } from '@graphql-codegen/cli';

const config: CodegenConfig = {
    // Use the schema from the backend
    schema: '../../internal/api/controlplane/graph/schema.graphqls',
    documents: ['src/**/*.tsx', 'src/**/*.ts', '!src/gql/**/*'],
    generates: {
        './src/gql/': {
            preset: 'client',
            plugins: [],
            presetConfig: {
                fragmentMasking: false,
            },
        },
        './src/gql/urql.ts': {
            plugins: ['typescript', 'typescript-operations', 'typescript-urql'],
            config: {
                withHooks: true,
                withComponent: false,
            },
        },
    },
    ignoreNoDocuments: true,
};

export default config;
