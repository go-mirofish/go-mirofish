"""
"""

import uuid
import json
import os
import threading
from datetime import datetime
from enum import Enum
from typing import Dict, Any, Optional, List
from dataclasses import dataclass, field

from ..config import Config
from ..utils.locale import t


class TaskStatus(str, Enum):
    PENDING = "pending"          # Waiting
    PROCESSING = "processing"    # In progress
    COMPLETED = "completed"      # Completed
    FAILED = "failed"            # Failed


@dataclass
class Task:
    task_id: str
    task_type: str
    status: TaskStatus
    created_at: datetime
    updated_at: datetime
    progress: int = 0              # Overall progress percentage 0-100
    message: str = ""              # Status message
    result: Optional[Dict] = None  # Task result
    error: Optional[str] = None    # Error information
    metadata: Dict = field(default_factory=dict)  # Extra metadata
    progress_detail: Dict = field(default_factory=dict)  # Detailed progress information
    
    def to_dict(self) -> Dict[str, Any]:
        return {
            "task_id": self.task_id,
            "task_type": self.task_type,
            "status": self.status.value,
            "created_at": self.created_at.isoformat(),
            "updated_at": self.updated_at.isoformat(),
            "progress": self.progress,
            "message": self.message,
            "progress_detail": self.progress_detail,
            "result": self.result,
            "error": self.error,
            "metadata": self.metadata,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "Task":
        status = data.get("status", TaskStatus.PENDING.value)
        if isinstance(status, str):
            status = TaskStatus(status)
        return cls(
            task_id=data["task_id"],
            task_type=data.get("task_type", ""),
            status=status,
            created_at=datetime.fromisoformat(data["created_at"]),
            updated_at=datetime.fromisoformat(data["updated_at"]),
            progress=data.get("progress", 0),
            message=data.get("message", ""),
            result=data.get("result"),
            error=data.get("error"),
            metadata=data.get("metadata", {}),
            progress_detail=data.get("progress_detail", {}),
        )


class TaskManager:
    """
    """

    TASKS_DIR = os.path.join(Config.UPLOAD_FOLDER, "tasks")
    
    _instance = None
    _lock = threading.Lock()
    
    def __new__(cls):
        if cls._instance is None:
            with cls._lock:
                if cls._instance is None:
                    cls._instance = super().__new__(cls)
                    cls._instance._tasks: Dict[str, Task] = {}
                    cls._instance._task_lock = threading.Lock()
                    cls._instance._ensure_tasks_dir()
        return cls._instance

    def _ensure_tasks_dir(self):
        os.makedirs(self.TASKS_DIR, exist_ok=True)

    def _task_path(self, task_id: str) -> str:
        return os.path.join(self.TASKS_DIR, f"{task_id}.json")

    def _save_task_to_disk(self, task: Task):
        self._ensure_tasks_dir()
        with open(self._task_path(task.task_id), "w", encoding="utf-8") as f:
            json.dump(task.to_dict(), f, ensure_ascii=False, indent=2)

    def _load_task_from_disk(self, task_id: str) -> Optional[Task]:
        task_path = self._task_path(task_id)
        if not os.path.exists(task_path):
            return None
        with open(task_path, "r", encoding="utf-8") as f:
            data = json.load(f)
        return Task.from_dict(data)

    def _load_all_tasks_from_disk(self) -> List[Task]:
        self._ensure_tasks_dir()
        tasks: List[Task] = []
        for filename in os.listdir(self.TASKS_DIR):
            if not filename.endswith(".json"):
                continue
            task_path = os.path.join(self.TASKS_DIR, filename)
            try:
                with open(task_path, "r", encoding="utf-8") as f:
                    tasks.append(Task.from_dict(json.load(f)))
            except Exception:
                continue
        return tasks
    
    def create_task(self, task_type: str, metadata: Optional[Dict] = None) -> str:
        """
        
        Args:
            
        Returns:
        """
        task_id = str(uuid.uuid4())
        now = datetime.now()
        
        task = Task(
            task_id=task_id,
            task_type=task_type,
            status=TaskStatus.PENDING,
            created_at=now,
            updated_at=now,
            metadata=metadata or {}
        )
        
        with self._task_lock:
            self._tasks[task_id] = task
            self._save_task_to_disk(task)
        
        return task_id
    
    def get_task(self, task_id: str) -> Optional[Task]:
        with self._task_lock:
            task = self._tasks.get(task_id)
            if task is not None:
                return task
            task = self._load_task_from_disk(task_id)
            if task is not None:
                self._tasks[task_id] = task
            return task
    
    def update_task(
        self,
        task_id: str,
        status: Optional[TaskStatus] = None,
        progress: Optional[int] = None,
        message: Optional[str] = None,
        result: Optional[Dict] = None,
        error: Optional[str] = None,
        progress_detail: Optional[Dict] = None
    ):
        """
        
        Args:
        """
        with self._task_lock:
            task = self._tasks.get(task_id)
            if task:
                task.updated_at = datetime.now()
                if status is not None:
                    task.status = status
                if progress is not None:
                    task.progress = progress
                if message is not None:
                    task.message = message
                if result is not None:
                    task.result = result
                if error is not None:
                    task.error = error
                if progress_detail is not None:
                    task.progress_detail = progress_detail
                self._save_task_to_disk(task)
    
    def complete_task(self, task_id: str, result: Dict):
        self.update_task(
            task_id,
            status=TaskStatus.COMPLETED,
            progress=100,
            message=t('progress.taskComplete'),
            result=result
        )
    
    def fail_task(self, task_id: str, error: str):
        self.update_task(
            task_id,
            status=TaskStatus.FAILED,
            message=t('progress.taskFailed'),
            error=error
        )
    
    def list_tasks(self, task_type: Optional[str] = None) -> list:
        with self._task_lock:
            for task in self._load_all_tasks_from_disk():
                self._tasks[task.task_id] = task
            tasks = list(self._tasks.values())
            if task_type:
                tasks = [t for t in tasks if t.task_type == task_type]
            return [t.to_dict() for t in sorted(tasks, key=lambda x: x.created_at, reverse=True)]
    
    def cleanup_old_tasks(self, max_age_hours: int = 24):
        from datetime import timedelta
        cutoff = datetime.now() - timedelta(hours=max_age_hours)
        
        with self._task_lock:
            old_ids = [
                tid for tid, task in self._tasks.items()
                if task.created_at < cutoff and task.status in [TaskStatus.COMPLETED, TaskStatus.FAILED]
            ]
            for tid in old_ids:
                del self._tasks[tid]
                try:
                    os.remove(self._task_path(tid))
                except FileNotFoundError:
                    pass
