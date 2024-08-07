/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { Button, makeStyles } from '@material-ui/core'

import { useCloudProviderProps } from 'hooks/useCloudProvider'

const useStyles = makeStyles({
  boxShadow: {
    padding: '24px',
    boxShadow: '0 8px 16px #3a3a441f, 0 16px 32px #5a5b6a1f',
  },
  aside: {
    width: '100%',
    height: 'fit-content',
    borderRadius: '4px',
    display: 'flex',
    flexDirection: 'column',
    justifyContent: 'flex-start',
    flex: '1 1 0',
    position: 'sticky',
    top: 10,

    '& h2': {
      fontSize: '14px',
      fontWeight: 500,
      margin: '0 0 10px 0',
      height: 'fit-content',
    },

    '& span': {
      fontSize: '13px',
    },

    '& button': {
      padding: '10px 20px',
      marginTop: '20px',
    },

    '@media (max-width: 1200px)': {
      position: 'relative',
      boxShadow: 'none',
      borderRadius: '0',
      padding: '0',
      flex: 'auto',
      marginBottom: '30px',

      '& button': {
        width: 'max-content',
      },
    },
  },
  asideSection: {
    padding: '12px 0',
    borderBottom: '1px solid #e0e0e0',

    '& span': {
      color: '#808080',
    },

    '& p': {
      margin: '5px 0 0 0',
      fontSize: '13px',
    },
  },
})

export const PostgresClusterInstallFormSidebar = ({
  state,
  handleCreate,
  disabled,
}: {
  state: useCloudProviderProps['initialState']
  handleCreate: () => void
  disabled: boolean
}) => {
  const classes = useStyles()

  return (
    <div className={classes.aside}>
      <div className={classes.boxShadow}>
        {state.patroni_cluster_name && (
          <div className={classes.asideSection}>
            <span>Cluster Name</span>
            <p>{state.patroni_cluster_name}</p>
          </div>
        )}
        <div className={classes.asideSection}>
          <span>Postgres version</span>
          <p>{state.version}</p>
        </div>
        <div className={classes.asideSection}>
          <span>Data directory</span>
          <p>{state.postgresql_data_dir}</p>
        </div>
        <Button
          variant="contained"
          color="primary"
          onClick={handleCreate}
          disabled={!state.patroni_cluster_name || disabled}
        >
          Install Postgres Cluster
        </Button>
      </div>
    </div>
  )
}
